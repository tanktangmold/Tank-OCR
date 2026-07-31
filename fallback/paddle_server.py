import sys
import os
import uvicorn
from fastapi import FastAPI, File, UploadFile
from fastapi.responses import JSONResponse
from paddleocr import PaddleOCR
import numpy as np
from PIL import Image
import io

app = FastAPI(title="Local Open-Source PaddleOCR API")

# Initialize PaddleOCR with Japanese and English support
# It will automatically download the lightweight models on first run
print("Initializing PaddleOCR (Japanese/English)...")
ocr = PaddleOCR(use_angle_cls=True, lang="japan")
print("PaddleOCR successfully initialized!")

@app.post("/api/ocr")
async def perform_ocr(file: UploadFile = File(...)):
    try:
        # Read uploaded image bytes
        contents = await file.read()
        image = Image.open(io.BytesIO(contents)).convert("RGB")
        
        # Convert PIL image to numpy array
        img_np = np.array(image)
        
        # Run PaddleOCR inference
        # structure: [[ [box], (text, confidence) ], ...]
        result = ocr.ocr(img_np, cls=True)
        
        # Format response to match the Go Local Fallback engine expectations
        formatted_result = []
        full_text = []
        
        if result and result[0]:
            for line in result[0]:
                box = line[0]        # [[x0, y0], [x1, y1], [x2, y2], [x3, y3]]
                text_info = line[1]  # ("text", confidence)
                
                text = text_info[0]
                confidence = float(text_info[1])
                
                formatted_result.append({
                    "text": text,
                    "confidence": confidence,
                    "box": box
                })
                full_text.append(text)
                
        return JSONResponse(content={
            "text": "\n".join(full_text),
            "result": formatted_result
        })
        
    except Exception as e:
        return JSONResponse(
            status_code=500,
            content={"detail": f"OCR processing failed: {str(e)}"}
        )

if __name__ == "__main__":
    # Run server locally on port 5000 (private loopback interface)
    uvicorn.run(app, host="127.0.0.1", port=5000)
