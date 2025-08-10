# Receiptify

A tool to print receipts with a thermal printer, enormously over-engineered.

## How it works

The project comes in 2 parts: a client app written in Go, and a server written in Python. The app designs templates in JSON which can be used to define the layout of a receipt. These templates can be reused, and also support Lua plugins to generate content at printing time. The server receives JSON templates, renders them as an image and then prints them via ESCPOS over USB.