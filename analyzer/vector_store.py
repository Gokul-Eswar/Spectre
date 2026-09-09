import os
import json

# Local-first embedding function definition:
# We instantiate SentenceTransformerEmbeddingFunction which runs a local transformer model.
# By default, we use 'all-MiniLM-L6-v2', a fast, high-performance, and lightweight model (under 120MB)
# that converts sentences/paragraphs into 384-dimensional dense vectors.
_default_ef = None


def get_embedding_function():
    global _default_ef
    if _default_ef is None:
        try:
            from chromadb.utils import embedding_functions

            _default_ef = embedding_functions.SentenceTransformerEmbeddingFunction(
                model_name="all-MiniLM-L6-v2"
            )
        except Exception:
            return None
    return _default_ef


def get_client(case_id):
    """
    Initializes and returns a persistent ChromaDB database client for a specific case.

    Parameters:
    - case_id (str): The unique ID of the investigation case.

    The database state is saved to a hidden folder '.vector_store' located inside the case's
    evidence directory (e.g. evidence_storage/<case_id>/.vector_store). This keeps case data
    isolated and local.
    """
    try:
        import chromadb
    except ImportError:
        return None

    persist_directory = os.path.join("evidence_storage", case_id, ".vector_store")
    os.makedirs(persist_directory, exist_ok=True)
    # Instantiate a persistent Chroma client. It writes database files to disk at the path.
    return chromadb.PersistentClient(path=persist_directory)


def index_evidence(case_id, evidence_files):
    """
    Indexes a list of evidence files into the case's vector store database for semantic retrieval.

    Parameters:
    - case_id (str): The unique ID of the case.
    - evidence_files (list of str): Paths to raw evidence files (e.g., ports.json, whois.json).

    Returns:
    - dict: A success status dictionary indicating the count of indexed files.
    """
    # 1. Resolve case-specific database client.
    client = get_client(case_id)
    if client is None:
        return {
            "status": "error",
            "error": "chromadb is not installed in Python environment",
            "indexed_count": 0,
        }

    ef = get_embedding_function()
    if ef is None:
        return {
            "status": "error",
            "error": "embedding function unavailable (sentence-transformers missing)",
            "indexed_count": 0,
        }

    # 2. Get or create the vector collection named 'evidence'.
    collection = client.get_or_create_collection(name="evidence", embedding_function=ef)

    ids = []
    documents = []
    metadatas = []

    # 3. Iterate over each provided file path and extract contents.
    for file_path in evidence_files:
        # Check if the file exists to prevent throwing runtime exceptions.
        if not os.path.exists(file_path):
            continue

        file_name = os.path.basename(file_path)

        # Operational security & layout check:
        # Skip hidden files/directories (starting with '.') or HTML visual reports.
        if file_name.startswith(".") or file_name == "report.html":
            continue

        # Read the raw file content. We use 'ignore' errors flag to handle binary files or
        # non-UTF-8 characters gracefully without crashing the analysis run.
        with open(file_path, "r", encoding="utf-8", errors="ignore") as f:
            content = f.read()

        # Chunk large evidence content into 500-character blocks with 50-character overlap
        chunk_size = 500
        overlap = 50
        chunks = []
        if len(content) <= chunk_size:
            chunks.append((content, 0))
        else:
            step = chunk_size - overlap
            for i in range(0, len(content), step):
                chunks.append((content[i : i + chunk_size], i))

        for idx, (chunk_text, offset) in enumerate(chunks):
            chunk_id = f"{file_name}#chunk_{idx}"
            ids.append(chunk_id)
            documents.append(chunk_text)
            metadatas.append(
                {"path": file_path, "name": file_name, "chunk": idx, "offset": offset}
            )

    # 4. If we gathered any valid files, upsert them to the database.
    if ids:
        collection.upsert(ids=ids, documents=documents, metadatas=metadatas)
        return {"status": "success", "indexed_count": len(ids)}

    return {"status": "success", "indexed_count": 0}


def search_evidence(case_id, query, n_results=3):
    """
    Performs a semantic vector search across the indexed evidence documents.
    """
    client = get_client(case_id)
    if client is None:
        return {
            "status": "error",
            "error": "chromadb is not installed in Python environment",
            "results": [],
        }

    ef = get_embedding_function()
    if ef is None:
        return {
            "status": "error",
            "error": "embedding function unavailable (sentence-transformers missing)",
            "results": [],
        }

    try:
        collection = client.get_collection(name="evidence", embedding_function=ef)
    except Exception:
        # Collection does not exist yet (no evidence indexed)
        return {"status": "success", "results": []}

    results = collection.query(query_texts=[query], n_results=n_results)

    formatted_results = []
    if results and "ids" in results and results["ids"] and len(results["ids"][0]) > 0:
        for i in range(len(results["ids"][0])):
            formatted_results.append(
                {
                    "id": results["ids"][0][i],
                    "content": results["documents"][0][i],
                    "metadata": results["metadatas"][0][i],
                    "distance": (
                        results["distances"][0][i]
                        if "distances" in results and results["distances"]
                        else 0.0
                    ),
                }
            )

    return {"status": "success", "results": formatted_results}
