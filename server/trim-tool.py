from escpos.printer import Usb

printer = Usb(0x04b8, 0x0202, 0, profile="TM-T88V")
printer.cut()