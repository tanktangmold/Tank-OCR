from __future__ import annotations

import sys
from pathlib import Path

import fitz


def main() -> int:
    if len(sys.argv) != 5:
        print(
            "usage: pdf_render.py INPUT.pdf OUTPUT_DIR PAGE_LIST MAX_DIMENSION",
            file=sys.stderr,
        )
        return 2

    input_path = Path(sys.argv[1])
    output_dir = Path(sys.argv[2])
    selected = [int(value) for value in sys.argv[3].split(",") if value]
    max_dimension = max(256, int(sys.argv[4]))

    document = fitz.open(input_path)
    try:
        page_numbers = selected or list(range(1, len(document) + 1))
        for page_number in page_numbers:
            if page_number < 1 or page_number > len(document):
                raise ValueError(
                    f"page {page_number} exceeds PDF page count {len(document)}"
                )

            page = document[page_number - 1]
            base_scale = 150.0 / 72.0
            rendered_width = page.rect.width * base_scale
            rendered_height = page.rect.height * base_scale
            scale = min(
                base_scale,
                max_dimension / max(page.rect.width, page.rect.height),
            )
            if max(rendered_width, rendered_height) <= max_dimension:
                scale = base_scale

            pixmap = page.get_pixmap(matrix=fitz.Matrix(scale, scale), alpha=False)
            output_path = output_dir / f"page-{page_number:06d}.png"
            pixmap.save(output_path)
    finally:
        document.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
