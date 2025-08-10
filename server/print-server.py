import json
import os
import qrcode
from flask import Flask, request, jsonify
from escpos.printer import Usb
from PIL import Image, ImageDraw, ImageFont

DEFAULT_FONT_PATH = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"
CANVAS_WIDTH = 512

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
    new_font = name + style + ext
    return os.path.join(dirname, new_font)

def wrap_text(draw, text, font, max_width):
    words = text.split()
    lines = []
    line = ""
    for word in words:
        test_line = line + (" " if line else "") + word
        bbox = draw.textbbox((0, 0), test_line, font=font)
        width = bbox[2] - bbox[0]
        if width <= max_width:
            line = test_line
        else:
            lines.append(line)
            line = word
    if line:
        lines.append(line)
    return lines

def render_receipt(template: list[dict], font_path=DEFAULT_FONT_PATH) -> Image.Image:
    MAX_HEIGHT = 10000
    img = Image.new("RGB", (CANVAS_WIDTH, MAX_HEIGHT), "white")
    draw = ImageDraw.Draw(img)
    y_offset = 0

    for component in template:
        ctype = component["type"]

        if ctype == "text":
            text = component.get("content", "")
            font_size = component.get("font_size", 14) * 2
            align = component.get("align", "left")
            bold = component.get("bold", False)
            italic = component.get("italic", False)
            underline = component.get("underline", False)

            font_path_using = get_font_path(font_path, bold=bold, italic=italic)
            try:
                font = ImageFont.truetype(font_path_using, font_size)
            except IOError:
                font = ImageFont.truetype(DEFAULT_FONT_PATH, font_size)

            for paragraph in text.split('\n'):
                lines = wrap_text(draw, paragraph, font, CANVAS_WIDTH - 40)
                for line in lines:
                    bbox = draw.textbbox((0, 0), line, font=font)
                    text_width = bbox[2] - bbox[0]
                    text_height = bbox[3] - bbox[1]

                    if align == "center":
                        x = (CANVAS_WIDTH - text_width) // 2
                    elif align == "right":
                        x = CANVAS_WIDTH - text_width - 20
                    else:
                        x = 20

                    draw.text((x, y_offset), line, font=font, fill="black")

                    if underline:
                        draw.line(
                            (x, y_offset + text_height + 4, x + text_width, y_offset + text_height + 4),
                            fill="black",
                            width=1,
                        )
                    y_offset += text_height + 5
                y_offset += 5

        elif ctype == "divider":
            y_offset += 10
            line_width = component.get("line_width", 1)
            draw.line((20, y_offset, CANVAS_WIDTH - 20, y_offset), fill="black", width=line_width)
            y_offset += 10 + line_width

        elif ctype == "qr":
            qr_content = component.get("content", "")
            fit = component.get("fit", None)  # None means not set
            scale = component.get("scale", None)  # None means not set
            align = component.get("align", "center")

            if not qr_content:
                continue

            # Create QR code image
            qr = qrcode.QRCode(
                version=1,
                error_correction=qrcode.constants.ERROR_CORRECT_L,
                box_size=10,
                border=2,
            )
            qr.add_data(qr_content)
            qr.make(fit=True)
            qr_img = qr.make_image(fill_color="black", back_color="white").convert("RGB")

            max_width = CANVAS_WIDTH - 40  # padding 20 px each side

            if fit is True:
                target_width = max_width
            elif scale is not None:
                if not (0 < scale <= 100):
                    raise ValueError("Scale must be between 1 and 100")
                target_width = int(max_width * (scale / 100))
            else:
                target_width = 200

            qr_img = qr_img.resize((target_width, target_width), Image.LANCZOS)

            if align == "center":
                x = (CANVAS_WIDTH - qr_img.width) // 2
            elif align == "right":
                x = CANVAS_WIDTH - qr_img.width - 20
            else:
                x = 20

            img.paste(qr_img, (x, y_offset))
            y_offset += qr_img.height + 10

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
