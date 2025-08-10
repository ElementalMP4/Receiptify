#!/bin/bash
set -e

/opt/receiptify/venv/bin/pip install --upgrade pip
/opt/receiptify/venv/bin/pip install -r %s/requirements.txt

exit 0
