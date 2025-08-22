from fastapi import FastAPI, Request
from pydantic import BaseModel
from sentence_transformers import SentenceTransformer
import uvicorn

app = FastAPI()
model = SentenceTransformer("sentence-transformers/all-MiniLM-L6-v2")

class EmbedRequest(BaseModel):
    text: str

@app.post("/embed")
async def embed(req: EmbedRequest):
    vec = model.encode(req.text, normalize_embeddings=True)
    return vec.tolist()

if __name__ == "__main__":
    uvicorn.run(app, host="127.0.0.1", port=8000)
