import json
import os
import io
import base64
import qrcode
from flask import Flask, request, jsonify
from escpos.printer import Usb
from PIL import Image, ImageDraw, ImageFont

DEFAULT_FONT_PATH = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"
CANVAS_WIDTH = 512
MAX_HEIGHT = 10000
MARGIN = 20
LINE_SPACING = 5

printer = Usb(0x04b8, 0x0202, 0, profile="TM-T88V")

def get_font_path(base_path, bold=False, italic=False):
    dirname, basename = os.path.split(base_path)
    name, ext = os.path.splitext(basename)
    style = ""
    if bold and italic:
        style = "-BoldOblique"
    elif bold:
        style = "-Bold"
    elif italic:
        style = "-Oblique"
    return os.path.join(dirname, name + style + ext)

def load_font(font_path, size, bold=False, italic=False):
    try:
        return ImageFont.truetype(get_font_path(font_path, bold, italic), size)
    except IOError:
        return ImageFont.truetype(DEFAULT_FONT_PATH, size)

def calculate_x(align, element_width, margin=MARGIN):
    if align == "center":
        return (CANVAS_WIDTH - element_width) // 2
    elif align == "right":
        return CANVAS_WIDTH - element_width - margin
    else:
        return margin

def wrap_text(draw, text, font, max_width):
    words = text.split()
    lines, line = [], ""
    for word in words:
        test_line = line + (" " if line else "") + word
        bbox = draw.textbbox((0, 0), test_line, font=font)
        if bbox[2] - bbox[0] <= max_width:
            line = test_line
        else:
            if line: lines.append(line)
            line = word
    if line:
        lines.append(line)
    return lines

def draw_text(draw, text, font, y_offset, align="left", underline=False):
    for paragraph in text.split("\n"):
        lines = wrap_text(draw, paragraph, font, CANVAS_WIDTH - 2*MARGIN)
        for line in lines:
            bbox = draw.textbbox((0, 0), line, font=font)
            width, height = bbox[2] - bbox[0], bbox[3] - bbox[1]
            x = calculate_x(align, width)
            draw.text((x, y_offset), line, font=font, fill="black")
            if underline:
                draw.line((x, y_offset + height + 2, x + width, y_offset + height + 2), fill="black", width=1)
            y_offset += height + max(LINE_SPACING, int(font.size*0.3))
        y_offset += LINE_SPACING
    return y_offset

def paste_image(base_img, element_img, y_offset, align="center", max_width=CANVAS_WIDTH - 2*MARGIN):
    aspect_ratio = element_img.height / element_img.width
    target_width = min(element_img.width, max_width)
    target_height = int(target_width * aspect_ratio)
    element_img = element_img.resize((target_width, target_height), Image.LANCZOS)
    x = calculate_x(align, target_width)
    base_img.paste(element_img, (x, y_offset))
    return y_offset + target_height + 10

def render_text_component(draw, component, y_offset):
    text = component.get("content", "")
    align = component.get("align", "left")
    bold = component.get("bold", False)
    italic = component.get("italic", False)
    underline = component.get("underline", False)

    font_size_raw = component.get("font_size", 14)
    if str(font_size_raw).lower() == "fit":
        size = 200
        while size > 10:
            font = load_font(DEFAULT_FONT_PATH, size, bold, italic)
            if draw.textbbox((0,0), text, font=font)[2] <= (CANVAS_WIDTH - 2*MARGIN):
                break
            size -= 2
        else:
            size = 14
        font = load_font(DEFAULT_FONT_PATH, size, bold, italic)
    else:
        try:
            size = int(font_size_raw)
        except (TypeError, ValueError):
            size = 14
        font = load_font(DEFAULT_FONT_PATH, size*2, bold, italic)

    return draw_text(draw, text, font, y_offset, align=align, underline=underline)

def render_divider_component(draw, component, y_offset):
    y_offset += 10
    line_width = component.get("line_width", 1)
    draw.line((MARGIN, y_offset, CANVAS_WIDTH - MARGIN, y_offset), fill="black", width=line_width)
    return y_offset + 10 + line_width

def render_qr_component(img, component, y_offset):
    content = component.get("content", "")
    if not content:
        return y_offset
    align = component.get("align", "center")
    fit = component.get("fit", None)
    scale = component.get("scale", None)

    qr = qrcode.QRCode(version=1, error_correction=qrcode.constants.ERROR_CORRECT_L, box_size=10, border=2)
    qr.add_data(content)
    qr.make(fit=True)
    qr_img = qr.make_image(fill_color="black", back_color="white").convert("RGB")

    max_width = CANVAS_WIDTH - 2*MARGIN
    if fit is True:
        target_width = max_width
    elif scale:
        target_width = int(max_width * (scale/100))
    else:
        target_width = 200

    qr_img = qr_img.resize((target_width, target_width), Image.LANCZOS)
    return paste_image(img, qr_img, y_offset, align=align)

def render_image_component(img, component, y_offset):
    b64_data = component.get("content", "")
    if not b64_data:
        return y_offset
    align = component.get("align", "center")
    fit = component.get("fit", False)
    scale = component.get("scale", None)
    pixel_width = component.get("width", None)

    try:
        pil_img = Image.open(io.BytesIO(base64.b64decode(b64_data))).convert("RGB")
    except Exception:
        return y_offset

    max_width = CANVAS_WIDTH - 2*MARGIN
    if fit:
        target_width = max_width
    elif pixel_width:
        target_width = min(pixel_width, max_width)
    elif scale:
        target_width = int(max_width * (scale/100))
    else:
        target_width = min(pil_img.width, max_width)

    aspect_ratio = pil_img.height / pil_img.width
    pil_img = pil_img.resize((target_width, int(target_width * aspect_ratio)), Image.LANCZOS)
    return paste_image(img, pil_img, y_offset, align=align)

COMPONENT_HANDLERS = {
    "text": render_text_component,
    "header": render_text_component,
    "macro": render_text_component,
    "divider": render_divider_component,
    "qr": render_qr_component,
    "image": render_image_component,
}

def render_receipt(template: list[dict], font_path=DEFAULT_FONT_PATH) -> Image.Image:
    img = Image.new("RGB", (CANVAS_WIDTH, MAX_HEIGHT), "white")
    draw = ImageDraw.Draw(img)
    y_offset = 0

    for component in template:
        ctype = component.get("type")
        handler = COMPONENT_HANDLERS.get(ctype)
        if handler:
            if ctype in ["qr", "image"]:
                y_offset = handler(img, component, y_offset)
            else:
                y_offset = handler(draw, component, y_offset)

    final_img = img.crop((0, 0, CANVAS_WIDTH, y_offset + 20))
    _, final_height = final_img.size
    if final_height > MAX_HEIGHT:
        raise ValueError("Receipt is too long.")
    return final_img

def print_receipt_image(receipt_image: Image.Image):
    printer.image(receipt_image)
    printer.cut()

app = Flask(__name__)

@app.route("/print-receipt", methods=["POST"])
def print_receipt():
    try:
        template = request.get_json(force=True)
        if not isinstance(template, list):
            return jsonify({"error": "Invalid template format: expected a list"}), 400
        receipt_img = render_receipt(template)
        print_receipt_image(receipt_img)
        return jsonify({"message": "Receipt printed successfully"}), 200
    except Exception as e:
        return jsonify({"error": str(e)}), 500

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)
