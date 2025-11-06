# Industrial Symbiosis Prototype - Backend

A Go-based backend with Python document parsing for industrial symbiosis matching using MCP orchestration and Gemini API.

## Architecture

- **Go Backend**: High-performance API server with MCP orchestration
- **Python Worker**: Document parsing service for PDF, DOCX, TXT files
- **PostgreSQL**: Database with JSONB for flexible schema
- **Local File Storage**: Simple file storage for uploaded documents
- **Gemini API**: LLM for extraction, classification, and reasoning via MCP

## Features

- Document upload and async processing
- Automated extraction of industry inputs/outputs/waste streams
- AI-powered waste classification using Gemini
- Intelligent matching engine with scoring
- Conversion requirement estimation
- Match reasoning generation
- RESTful API endpoints

## Project Structure

```
.
├── main.go                 # Application entry point
├── models.go              # Data structures
├── database.go            # PostgreSQL operations
├── storage.go             # Local file storage operations
├── mcp_client.go          # MCP/Gemini API client
├── handlers.go            # HTTP request handlers
├── processor.go           # Document processing pipeline
├── go.mod                 # Go dependencies
├── python_worker/
│   ├── app.py            # Python Flask worker
│   ├── document_parser.py # Document parsing logic
│   └── requirements.txt   # Python dependencies
├── .env.example          # Environment variables template
└── README.md             # This file
```

## Prerequisites

- **Go 1.21+** - [Download](https://golang.org/dl/)
- **Python 3.11+** - [Download](https://www.python.org/downloads/)
- **PostgreSQL 12+** - [Download](https://www.postgresql.org/download/)
- **Gemini API Key** - [Get one here](https://makersuite.google.com/app/apikey)

## Local Development Setup

### Step 1: Verify Prerequisites

```bash
# Check Go installation
go version
# Should show: go version go1.21.x or higher

# Check Python installation
python3 --version
# Should show: Python 3.11.x or higher

# Check PostgreSQL installation
psql --version
# Should show: psql (PostgreSQL) 12.x or higher
```

### Step 2: Setup PostgreSQL Database

```bash
# Option A: Using createdb command
createdb industrial_symbiosis

# Option B: Using psql
psql -U postgres
CREATE DATABASE industrial_symbiosis;
\q

# Verify database was created
psql -U postgres -l | grep industrial_symbiosis
```

If you get authentication errors, you may need to:

### Option C: Run PostgreSQL in Docker (recommended if you have Docker)

If you have Docker Desktop installed on Windows you can run a Postgres container with the defaults used by this project.

1. Start the database using Docker Compose (run from the project root):

```powershell
docker compose up -d
```

2. The compose file exposes Postgres on localhost:5432 with the following defaults (matching `database.go`):

- user: `postgres`
- password: `postgres`
- database: `industrial_symbiosis`

3. Use the `.env.example` to create a `.env` file and ensure `DATABASE_URL` is set (the default matches the container):

```powershell
copy .env.example .env
# then edit .env if you want different credentials
```

4. Verify Postgres is reachable from your host (PowerShell):

```powershell
# Wait a few seconds for DB to initialize, then test a connection with psql (if installed)
psql "host=localhost port=5432 user=postgres password=postgres dbname=industrial_symbiosis"
```

Note: If you prefer to run both the Go backend and Python worker in Docker as services, we can extend the `docker-compose.yml` to build and run them. For now the compose file only starts Postgres and exposes it to the host so you can continue running the backend and worker locally while using a containerized database.
```bash
# On macOS (if using Homebrew):
brew services start postgresql

# On Linux:
sudo systemctl start postgresql
sudo -u postgres psql

# On Windows:
# Start PostgreSQL from Services app or pgAdmin
```

### Step 3: Clone/Setup Project

```bash
# Navigate to your project directory
cd /path/to/your/project

# Your directory structure should look like the Project Structure above
# Ensure you have storage.go (local version, not MinIO version)
```

### Step 4: Configure Environment Variables

Create a `.env` file in the project root:

```bash
# Copy the example
cp .env.example .env

# Edit the .env file with your settings
```

Your `.env` file should contain:

```bash
# Server Configuration
PORT=8080

# Database Configuration
# Update 'password' with your actual PostgreSQL password
DATABASE_URL=host=localhost port=5432 user=postgres password=postgres dbname=industrial_symbiosis sslmode=disable

# File Storage Configuration
UPLOAD_DIR=./uploads

# Python Worker Configuration
PYTHON_WORKER_URL=http://localhost:5000

# Gemini API Configuration
# IMPORTANT: Replace with your actual API key
GEMINI_API_KEY=your_actual_gemini_api_key_here
```

### Step 5: Setup Go Backend

```bash
# Install Go dependencies
go mod download

# Verify all dependencies are installed
go mod tidy

# This should complete without errors
```

If you see errors about missing packages, run:
```bash
go get github.com/gin-gonic/gin
go get github.com/google/uuid
go get github.com/joho/godotenv
go get github.com/lib/pq
```

### Step 6: Setup Python Worker

```bash
# Navigate to python_worker directory
cd python_worker

# Create virtual environment
python3 -m venv venv

# Activate virtual environment
# On macOS/Linux:
source venv/bin/activate

# On Windows:
venv\Scripts\activate

# You should see (venv) in your terminal prompt

# Install Python dependencies
pip install -r requirements.txt

# Verify installation
pip list
# Should show Flask, PyPDF2, python-docx, requests, gunicorn
```

### Step 7: Run the Application

You'll need **TWO terminal windows** - one for Go backend, one for Python worker.

#### Terminal 1: Start Python Worker

```bash
# Make sure you're in the python_worker directory
cd python_worker

# Activate virtual environment if not already active
source venv/bin/activate  # macOS/Linux
# OR
venv\Scripts\activate     # Windows

# Run the worker
python app.py

# You should see:
# * Running on http://0.0.0.0:5000
# * Running on http://127.0.0.1:5000
```

#### Terminal 2: Start Go Backend

```bash
# Make sure you're in the project root directory
# (where main.go is located)

# Run the backend
go run .

# You should see:
# Server starting on port 8080
# (and database/storage initialization messages)
```

### Step 8: Test the Application

Open a **third terminal** to test:

```bash
# Test Go backend health
curl http://localhost:8080/health
# Expected: {"status":"healthy"}

# Test Python worker health
curl http://localhost:5000/health
# Expected: {"status":"healthy"}
```

#### Create a Test Document

```bash
# Create a test company profile
cat > test_company.txt << 'EOF'
Steel Rolling Mill A
Location: 12.34, 56.78

Inputs:
- scrap metal (500 tons/month)
- coal
- cooling water

Outputs:
- waste slag (200 tons/month)
- cooling water discharge (liquid, 500 litres/day)
- steel rods (primary product)
EOF
```

#### Upload the Document

```bash
# Upload the test document
curl -X POST http://localhost:8080/api/v1/upload \
  -F "file=@test_company.txt"

# Expected response:
# {
#   "task_id": "some-uuid-here",
#   "file_url": "/absolute/path/to/uploads/filename",
#   "status": "pending"
# }
```

#### Check Processing Status

```bash
# Replace 'your-task-id' with the actual task_id from upload response
curl http://localhost:8080/api/v1/tasks/your-task-id

# Expected response (when completed):
# {
#   "id": "your-task-id",
#   "status": "completed",
#   "profile_id": "profile-uuid",
#   "result": {...}
# }
```

#### View All Profiles

```bash
# List all industry profiles
curl http://localhost:8080/api/v1/profiles

# Expected response:
# {
#   "count": 1,
#   "profiles": [
#     {
#       "id": "uuid",
#       "name": "Steel Rolling Mill A",
#       "location": {"lat": 12.34, "lng": 56.78},
#       "inputs": [...],
#       "outputs": [...]
#     }
#   ]
# }
```

#### View Matches (After uploading multiple profiles)

```bash
# Replace 'profile-id' with actual profile ID
curl http://localhost:8080/api/v1/profiles/profile-id/matches

# Expected response:
# {
#   "profile_id": "uuid",
#   "matches": [...]
# }
```

## API Endpoints Reference

### 1. Upload Document
```bash
POST /api/v1/upload
Content-Type: multipart/form-data

curl -X POST http://localhost:8080/api/v1/upload \
  -F "file=@company_profile.pdf"
```

### 2. Get Task Status
```bash
GET /api/v1/tasks/:task_id

curl http://localhost:8080/api/v1/tasks/{task_id}
```

### 3. Get Profile
```bash
GET /api/v1/profiles/:profile_id

curl http://localhost:8080/api/v1/profiles/{profile_id}
```

### 4. Get Matches for Profile
```bash
GET /api/v1/profiles/:profile_id/matches

curl http://localhost:8080/api/v1/profiles/{profile_id}/matches
```

### 5. Confirm Match
```bash
POST /api/v1/matches/:match_id/confirm

curl -X POST http://localhost:8080/api/v1/matches/{match_id}/confirm
```

### 6. List All Profiles
```bash
GET /api/v1/profiles

curl http://localhost:8080/api/v1/profiles
```

## Troubleshooting

### Issue: "dial tcp [::1]:5432: connect: connection refused" (PostgreSQL)

**Solution:**
```bash
# Check if PostgreSQL is running
pg_isready

# If not running:
# macOS (Homebrew):
brew services start postgresql

# Linux:
sudo systemctl start postgresql
sudo systemctl status postgresql

# Windows:
# Start PostgreSQL service from Services app
```

### Issue: "password authentication failed for user postgres"

**Solution:**
```bash
# Update DATABASE_URL in .env with correct password
DATABASE_URL=host=localhost port=5432 user=postgres password=YOUR_PASSWORD dbname=industrial_symbiosis sslmode=disable

# Or create a new user:
psql -U postgres
CREATE USER your_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE industrial_symbiosis TO your_user;
\q
```

### Issue: "bind: address already in use" (Port 8080 or 5000 in use)

**Solution:**
```bash
# Find what's using the port:
# macOS/Linux:
lsof -i :8080
lsof -i :5000

# Windows:
netstat -ano | findstr :8080
netstat -ano | findstr :5000

# Option 1: Kill the process using the port
# Option 2: Change port in .env
PORT=8081  # for Go backend

# For Python worker, edit python_worker/app.py:
port = int(os.getenv('PORT', 5001))
# Then update .env:
PYTHON_WORKER_URL=http://localhost:5001
```

### Issue: Python "ModuleNotFoundError"

**Solution:**
```bash
# Make sure virtual environment is activated
source venv/bin/activate  # macOS/Linux
venv\Scripts\activate     # Windows

# Reinstall dependencies
pip install -r requirements.txt

# If still failing, try upgrading pip:
pip install --upgrade pip
pip install -r requirements.txt
```

### Issue: Go "package not found" errors

**Solution:**
```bash
# Clean and reinstall dependencies
go clean -modcache
go mod download
go mod tidy

# If specific package missing:
go get github.com/gin-gonic/gin
go get github.com/lib/pq
```

### Issue: "GEMINI_API_KEY not set"

**Solution:**
1. Get API key from https://makersuite.google.com/app/apikey
2. Add to `.env` file:
   ```bash
   GEMINI_API_KEY=your_actual_key_here
   ```
3. Restart the Go backend

### Issue: Upload directory not writable

**Solution:**
```bash
# Create uploads directory with correct permissions
mkdir -p ./uploads
chmod 755 ./uploads

# On Windows, ensure folder isn't read-only
```

### Issue: Python worker not responding

**Solution:**
```bash
# Check if Python worker is running
curl http://localhost:5000/health

# Check Python worker logs in Terminal 1
# Common issues:
# - Port already in use (change port)
# - Virtual environment not activated
# - Missing dependencies (reinstall)

# Restart the worker:
# Ctrl+C to stop
python app.py  # Start again
```

## Project Checklist

✅ **File Structure:**
- [ ] All Go files in root directory (main.go, models.go, database.go, storage.go, mcp_client.go, handlers.go, processor.go)
- [ ] go.mod file exists
- [ ] python_worker directory with app.py, document_parser.py, requirements.txt
- [ ] .env file created and configured
- [ ] storage.go is the LOCAL version (not MinIO version)

✅ **Prerequisites Installed:**
- [ ] Go 1.21+
- [ ] Python 3.11+
- [ ] PostgreSQL running

✅ **Configuration:**
- [ ] PostgreSQL database created
- [ ] .env file with correct DATABASE_URL
- [ ] Gemini API key added to .env
- [ ] Python virtual environment created and activated

✅ **Running:**
- [ ] Python worker running (Terminal 1)
- [ ] Go backend running (Terminal 2)
- [ ] Health checks passing

## Next Steps

1. **Upload multiple company profiles** to test matching
2. **Create sample documents** in different formats (PDF, DOCX, TXT)
3. **Test the complete workflow**: Upload → Parse → Extract → Match
4. **Review matches** and confirm them
5. **Integrate with frontend** (when ready)

## Development Tips

- **Keep both terminals running** - You need both Python worker and Go backend
- **Check logs** - Both terminals show useful debug information
- **Test incrementally** - Upload one document, verify it works, then upload more
- **Use curl or Postman** - Test API endpoints before building frontend
- **Monitor database** - Use `psql` to inspect data:
  ```bash
  psql -U postgres industrial_symbiosis
  SELECT * FROM industry_profiles;
  SELECT * FROM match_recommendations;
  \q
  ```

## Getting Gemini API Key

1. Go to https://makersuite.google.com/app/apikey
2. Sign in with Google account
3. Click "Create API Key"
4. Copy the key
5. Add to `.env` file:
   ```bash
   GEMINI_API_KEY=your_copied_key_here
   ```

## Support

For issues or questions:
1. Check the Troubleshooting section above
2. Review Go backend logs (Terminal 2)
3. Review Python worker logs (Terminal 1)
4. Check PostgreSQL is running and accessible
5. Verify all files are in correct locations per Project Structure

## License

MIT#   I n d u s t r i a l _ S y m b i o s i s _ P r o t o t y p e 
 
 #   I n d u s t r i a l _ S y m b i o s i s _ P r o t o t y p e 
 
 