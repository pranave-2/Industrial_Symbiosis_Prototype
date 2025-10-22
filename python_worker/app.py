from flask import Flask, request, jsonify
import os
import uuid
import requests
from datetime import datetime
from document_parser import DocumentParser

app = Flask(__name__)
parser = DocumentParser()

@app.route('/health', methods=['GET'])
def health():
    return jsonify({"status": "healthy"}), 200

@app.route('/parse', methods=['POST'])
def parse_document():
    """Parse uploaded document and extract industry profile"""
    try:
        data = request.json
        file_url = data.get('file_url')
        filename = data.get('filename')
        
        if not file_url or not filename:
            return jsonify({"error": "Missing file_url or filename"}), 400
        
        # Download file
        local_path = download_file(file_url, filename)
        
        # Parse document
        profile_data = parser.parse(local_path)
        
        # Clean up
        if os.path.exists(local_path):
            os.remove(local_path)
        
        # Create industry profile structure
        profile = {
            "id": str(uuid.uuid4()),
            "name": profile_data.get("name", "Unknown Company"),
            "location": profile_data.get("location", {"lat": 0.0, "lng": 0.0}),
            "inputs": profile_data.get("inputs", []),
            "outputs": profile_data.get("outputs", []),
            "created_at": datetime.utcnow().isoformat(),
            "updated_at": datetime.utcnow().isoformat()
        }
        
        return jsonify({"profile": profile}), 200
        
    except Exception as e:
        return jsonify({"error": str(e)}), 500

def download_file(url, filename):
    """Download file from URL to local temp directory"""
    temp_dir = "/tmp/industrial_symbiosis"
    os.makedirs(temp_dir, exist_ok=True)
    
    local_path = os.path.join(temp_dir, filename)
    
    response = requests.get(url, stream=True)
    response.raise_for_status()
    
    with open(local_path, 'wb') as f:
        for chunk in response.iter_content(chunk_size=8192):
            f.write(chunk)
    
    return local_path

if __name__ == '__main__':
    port = int(os.getenv('PORT', 5000))
    app.run(host='0.0.0.0', port=port, debug=False)