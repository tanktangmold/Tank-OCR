from __future__ import annotations

import json
import sys
from pathlib import Path

import fitz


def add_invisible_text(page: fitz.Page, ocr_page: dict) -> None:
    image_width = float(ocr_page.get("width") or 0)
    image_height = float(ocr_page.get("height") or 0)
    if image_width <= 0 or image_height <= 0:
        return

    scale_x = page.rect.width / image_width
    scale_y = page.rect.height / image_height

    for block in ocr_page.get("blocks") or []:
        for line in block.get("lines") or []:
            text = (line.get("text") or "").strip()
            box = line.get("bounding_box")
            if not text or not box:
                continue

            x0 = max(0.0, float(box.get("x", 0)) * scale_x)
            y0 = max(0.0, float(box.get("y", 0)) * scale_y)
            x1 = min(page.rect.width, x0 + float(box.get("width", 0)) * scale_x)
            y1 = min(page.rect.height, y0 + float(box.get("height", 0)) * scale_y)
            if x1 <= x0 or y1 <= y0:
                continue

            font_size = max(4.0, min(24.0, (y1 - y0) * 0.8))
            page.insert_text(
                fitz.Point(x0, y1),
                text,
                fontname="japan",
                fontsize=font_size,
                render_mode=3,
                overlay=True,
            )


def main() -> int:
    if len(sys.argv) != 4:
        print("usage: pdf_overlay.py INPUT.pdf OUTPUT.pdf OCR_RESULT.json", file=sys.stderr)
        return 2

    input_path, output_path, result_path = map(Path, sys.argv[1:])
    result = json.loads(result_path.read_text(encoding="utf-8"))
    pages_by_number = {
        int(page["page_number"]): page for page in result.get("pages") or []
    }

    document = fitz.open(input_path)
    try:
        for page_number, ocr_page in pages_by_number.items():
            if 1 <= page_number <= len(document):
                add_invisible_text(document[page_number - 1], ocr_page)
        document.save(output_path, garbage=3, deflate=True)
    finally:
        document.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
