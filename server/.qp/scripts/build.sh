#!/bin/bash
set -e

python3 -m venv venv
venv/bin/pip3 install -r requirements.txt

exit 0