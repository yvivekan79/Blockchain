package api

import (
        "fmt"
        "net/http"

        "github.com/gin-gonic/gin"
)

// SwaggerSpec represents the OpenAPI specification
type SwaggerSpec struct {
        OpenAPI    string                 `json:"openapi"`
        Info       SwaggerInfo            `json:"info"`
        Servers    []SwaggerServer        `json:"servers"`
        Paths      map[string]interface{} `json:"paths"`
        Components SwaggerComponents      `json:"components"`
}

type SwaggerInfo struct {
        Title          string `json:"title"`
        Description    string `json:"description"`
        Version        string `json:"version"`
        Contact        SwaggerContact `json:"contact"`
        License        SwaggerLicense `json:"license"`
}

type SwaggerContact struct {
        Name  string `json:"name"`
        URL   string `json:"url"`
        Email string `json:"email"`
}

type SwaggerLicense struct {
        Name string `json:"name"`
        URL  string `json:"url"`
}

type SwaggerServer struct {
        URL         string `json:"url"`
        Description string `json:"description"`
}

type SwaggerComponents struct {
        Schemas map[string]interface{} `json:"schemas"`
}

// ServeSwaggerJSON serves the OpenAPI JSON specification
func (h *Handlers) ServeSwaggerJSON(c *gin.Context) {
        host := c.Request.Host
        if host == "" {
                host = "localhost:5000"
        }

        spec := SwaggerSpec{
                OpenAPI: "3.0.3",
                Info: SwaggerInfo{
                        Title:       "LSCC Blockchain API",
                        Description: "Advanced Layered Sharding with Cross-Channel Consensus (LSCC) blockchain research platform providing comprehensive API-driven blockchain infrastructure for academic and development purposes.",
                        Version:     "1.0.0",
                        Contact: SwaggerContact{
                                Name:  "LSCC Blockchain Team",
                                URL:   fmt.Sprintf("http://%s", host),
                                Email: "contact@lscc-blockchain.org",
                        },
                        License: SwaggerLicense{
                                Name: "MIT",
                                URL:  "https://opensource.org/licenses/MIT",
                        },
                },
                Servers: []SwaggerServer{
                        {
                                URL:         fmt.Sprintf("http://%s", host),
                                Description: "LSCC Blockchain API Server",
                        },
                },
                Paths:      generateSwaggerPaths(),
                Components: generateSwaggerComponents(),
        }

        c.Header("Access-Control-Allow-Origin", "*")
        c.JSON(http.StatusOK, spec)
}

// ServeSwaggerUI serves the Swagger UI interface
func (h *Handlers) ServeSwaggerUI(c *gin.Context) {
        host := c.Request.Host
        if host == "" {
                host = "localhost:5000"
        }

        html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>LSCC Blockchain API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *, *:before, *:after {
            box-sizing: inherit;
        }
        body {
            margin:0;
            background: #fafafa;
        }
        .swagger-ui .topbar {
            background-color: #1f2937;
        }
        .swagger-ui .topbar .download-url-wrapper .select-label {
            color: #ffffff;
        }
        .swagger-ui .topbar .download-url-wrapper input[type=text] {
            border: 2px solid #3b82f6;
        }
        .custom-header {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            padding: 20px;
            text-align: center;
            margin-bottom: 20px;
        }
        .custom-header h1 {
            margin: 0;
            font-size: 2.5em;
            font-weight: 300;
        }
        .custom-header p {
            margin: 10px 0 0 0;
            font-size: 1.1em;
            opacity: 0.9;
        }
        .stats-bar {
            background: #ffffff;
            border: 1px solid #e2e8f0;
            border-radius: 8px;
            padding: 15px;
            margin: 0 20px 20px 20px;
            display: flex;
            justify-content: space-around;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .stat-item {
            text-align: center;
        }
        .stat-value {
            font-size: 1.8em;
            font-weight: bold;
            color: #3b82f6;
        }
        .stat-label {
            font-size: 0.9em;
            color: #64748b;
            margin-top: 5px;
        }
    </style>
</head>
<body>
    <div class="custom-header">
        <h1>🔗 LSCC Blockchain API</h1>
        <p>Advanced Layered Sharding with Cross-Channel Consensus</p>
    </div>
    
    <div class="stats-bar">
        <div class="stat-item">
            <div class="stat-value" id="blockHeight">Loading...</div>
            <div class="stat-label">Block Height</div>
        </div>
        <div class="stat-item">
            <div class="stat-value" id="totalTx">Loading...</div>
            <div class="stat-label">Total Transactions</div>
        </div>
        <div class="stat-item">
            <div class="stat-value" id="activeShards">Loading...</div>
            <div class="stat-label">Active Shards</div>
        </div>
        <div class="stat-item">
            <div class="stat-value" id="consensusRounds">Loading...</div>
            <div class="stat-label">Consensus Rounds</div>
        </div>
        <div class="stat-item">
            <div class="stat-value" id="networkPeers">Loading...</div>
            <div class="stat-label">Network Peers</div>
        </div>
    </div>

    <div id="swagger-ui"></div>

    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            // Initialize Swagger UI
            const ui = SwaggerUIBundle({
                url: 'http://%s/api/swagger.json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                tryItOutEnabled: true,
                requestInterceptor: function(req) {
                    req.headers['Content-Type'] = 'application/json';
                    return req;
                }
            });

            // Load live blockchain stats
            function updateStats() {
                fetch('/api/v1/blockchain/info')
                    .then(response => response.json())
                    .then(data => {
                        document.getElementById('blockHeight').textContent = data.chain_height || '0';
                        document.getElementById('totalTx').textContent = data.total_transactions || '0';
                    })
                    .catch(err => {
                        document.getElementById('blockHeight').textContent = 'N/A';
                        document.getElementById('totalTx').textContent = 'N/A';
                    });

                fetch('/api/v1/shards/')
                    .then(response => response.json())
                    .then(data => {
                        document.getElementById('activeShards').textContent = data.shards ? data.shards.length : '0';
                    })
                    .catch(err => {
                        document.getElementById('activeShards').textContent = 'N/A';
                    });

                fetch('/api/v1/consensus/status')
                    .then(response => response.json())
                    .then(data => {
                        document.getElementById('consensusRounds').textContent = data.current_round || '0';
                    })
                    .catch(err => {
                        document.getElementById('consensusRounds').textContent = 'N/A';
                    });

                fetch('/api/v1/network/peers')
                    .then(response => response.json())
                    .then(data => {
                        document.getElementById('networkPeers').textContent = data.peer_count || '0';
                    })
                    .catch(err => {
                        document.getElementById('networkPeers').textContent = 'N/A';
                    });
            }

            // Update stats immediately and then every 5 seconds
            updateStats();
            setInterval(updateStats, 5000);
        };
    </script>
</body>
</html>`, host)

        c.Header("Content-Type", "text/html; charset=utf-8")
        c.String(http.StatusOK, html)
}

// generateSwaggerPaths generates the OpenAPI paths specification
func generateSwaggerPaths() map[string]interface{} {
        paths := make(map[string]interface{})

        // Health endpoint
        paths["/health"] = map[string]interface{}{
                "get": map[string]interface{}{
                        "tags":        []string{"System"},
                        "summary":     "Health Check",
                        "description": "Check the health status of the LSCC blockchain node",
                        "responses": map[string]interface{}{
                                "200": map[string]interface{}{
                                        "description": "Node is healthy",
                                        "content": map[string]interface{}{
                                                "application/json": map[string]interface{}{
                                                        "schema": map[string]interface{}{
                                                                "type": "object",
                                                                "properties": map[string]interface{}{
                                                                        "status":    map[string]interface{}{"type": "string", "example": "healthy"},
                                                                        "timestamp": map[string]interface{}{"type": "string", "format": "date-time"},
                                                                        "version":   map[string]interface{}{"type": "string", "example": "1.0.0"},
                                                                },
                                                        },
                                                },
                                        },
                                },
                        },
                },
        }

        // Blockchain endpoints
        paths["/api/v1/blockchain/info"] = map[string]interface{}{
                "get": map[string]interface{}{
                        "tags":        []string{"Blockchain"},
                        "summary":     "Get Blockchain Information",
                        "description": "Retrieve comprehensive information about the blockchain state including block height, transaction count, and performance metrics",
                        "responses": map[string]interface{}{
                                "200": map[string]interface{}{
                                        "description": "Blockchain information retrieved successfully",
                                        "content": map[string]interface{}{
                                                "application/json": map[string]interface{}{
                                                        "schema": map[string]interface{}{
                                                                "$ref": "#/components/schemas/BlockchainInfo",
                                                        },
                                                },
                                        },
                                },
                        },
                },
        }

        paths["/api/v1/blockchain/blocks"] = map[string]interface{}{
                "get": map[string]interface{}{
                        "tags":        []string{"Blockchain"},
                        "summary":     "Get Blocks",
                        "description": "Retrieve a list of blocks with pagination support",
                        "parameters": []interface{}{
                                map[string]interface{}{
                                        "name":        "limit",
                                        "in":          "query",
                                        "description": "Number of blocks to return (max 100)",
                                        "schema":      map[string]interface{}{"type": "integer", "default": 10, "maximum": 100},
                                },
                                map[string]interface{}{
                                        "name":        "offset",
                                        "in":          "query", 
                                        "description": "Number of blocks to skip",
                                        "schema":      map[string]interface{}{"type": "integer", "default": 0},
                                },
                        },
                        "responses": map[string]interface{}{
                                "200": map[string]interface{}{
                                        "description": "Blocks retrieved successfully",
                                        "content": map[string]interface{}{
                                                "application/json": map[string]interface{}{
                                                        "schema": map[string]interface{}{
                                                                "type": "object",
                                                                "properties": map[string]interface{}{
                                                                        "blocks": map[string]interface{}{
                                                                                "type":  "array",
                                                                                "items": map[string]interface{}{"$ref": "#/components/schemas/Block"},
                                                                        },
                                                                        "total_count": map[string]interface{}{"type": "integer"},
                                                                        "limit":       map[string]interface{}{"type": "integer"},
                                                                        "offset":      map[string]interface{}{"type": "integer"},
                                                                },
                                                        },
                                                },
                                        },
                                },
                        },
                },
        }

        // Transaction endpoints
        paths["/api/v1/transactions"] = map[string]interface{}{
                "post": map[string]interface{}{
                        "tags":        []string{"Transactions"},
                        "summary":     "Submit Transaction",
                        "description": "Submit a new transaction to the blockchain network",
                        "requestBody": map[string]interface{}{
                                "required":    true,
                                "description": "Transaction data",
                                "content": map[string]interface{}{
                                        "application/json": map[string]interface{}{
                                                "schema": map[string]interface{}{
                                                        "$ref": "#/components/schemas/TransactionRequest",
                                                },
                                        },
                                },
                        },
                        "responses": map[string]interface{}{
                                "201": map[string]interface{}{
                                        "description": "Transaction submitted successfully",
                                        "content": map[string]interface{}{
                                                "application/json": map[string]interface{}{
                                                        "schema": map[string]interface{}{
                                                                "$ref": "#/components/schemas/TransactionResponse",
                                                        },
                                                },
                                        },
                                },
                                "400": map[string]interface{}{
                                        "description": "Invalid transaction data",
                                        "content": map[string]interface{}{
                                                "application/json": map[string]interface{}{
                                                        "schema": map[string]interface{}{
                                                                "$ref": "#/components/schemas/Error",
                                                        },
                                                },
                                        },
                                },
                        },
                },
                "get": map[string]interface{}{
                        "tags":        []string{"Transactions"},
                        "summary":     "Get Transactions",
                        "description": "Retrieve a list of transactions with filtering and pagination",
                        "parameters": []interface{}{
                                map[string]interface{}{
                                        "name":        "limit",
                                        "in":          "query",
                                        "description": "Number of transactions to return",
                                        "schema":      map[string]interface{}{"type": "integer", "default": 10},
                                },
                                map[string]interface{}{
                                        "name":        "status",
                                        "in":          "query",
                                        "description": "Filter by transaction status",
                                        "schema":      map[string]interface{}{"type": "string", "enum": []string{"pending", "confirmed", "failed"}},
                                },
                        },
                        "responses": map[string]interface{}{
                                "200": map[string]interface{}{
                                        "description": "Transactions retrieved successfully",
                                },
                        },
                },
        }

        // Consensus endpoints
        paths["/api/v1/consensus/status"] = map[string]interface{}{
                "get": map[string]interface{}{
                        "tags":        []string{"Consensus"},
                        "summary":     "Get Consensus Status",
                        "description": "Retrieve current consensus algorithm status and performance metrics",
                        "responses": map[string]interface{}{
                                "200": map[string]interface{}{
                                        "description": "Consensus status retrieved successfully",
                                        "content": map[string]interface{}{
                                                "application/json": map[string]interface{}{
                                                        "schema": map[string]interface{}{
                                                                "$ref": "#/components/schemas/ConsensusStatus",
                                                        },
                                                },
                                        },
                                },
                        },
                },
        }

        // Network endpoints
        paths["/api/v1/network/peers"] = map[string]interface{}{
                "get": map[string]interface{}{
                        "tags":        []string{"Network"},
                        "summary":     "Get Network Peers",
                        "description": "Retrieve information about connected network peers",
                        "responses": map[string]interface{}{
                                "200": map[string]interface{}{
                                        "description": "Network peers retrieved successfully",
                                        "content": map[string]interface{}{
                                                "application/json": map[string]interface{}{
                                                        "schema": map[string]interface{}{
                                                                "$ref": "#/components/schemas/NetworkPeers",
                                                        },
                                                },
                                        },
                                },
                        },
                },
        }

        // Sharding endpoints
        paths["/api/v1/shards"] = map[string]interface{}{
                "get": map[string]interface{}{
                        "tags":        []string{"Sharding"},
                        "summary":     "Get Shards Information",
                        "description": "Retrieve information about all blockchain shards",
                        "responses": map[string]interface{}{
                                "200": map[string]interface{}{
                                        "description": "Shards information retrieved successfully",
                                },
                        },
                },
        }

        // Consensus Comparator endpoints
        paths["/api/v1/comparator/start"] = map[string]interface{}{
                "post": map[string]interface{}{
                        "tags":        []string{"Research & Testing"},
                        "summary":     "Start Consensus Comparison",
                        "description": "Start a performance comparison between different consensus algorithms",
                        "requestBody": map[string]interface{}{
                                "required": true,
                                "content": map[string]interface{}{
                                        "application/json": map[string]interface{}{
                                                "schema": map[string]interface{}{
                                                        "type": "object",
                                                        "properties": map[string]interface{}{
                                                                "algorithms":     map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string", "enum": []string{"pow", "pos", "pbft", "ppbft", "lscc"}}},
                                                                "duration":       map[string]interface{}{"type": "integer", "description": "Test duration in seconds"},
                                                                "transaction_rate": map[string]interface{}{"type": "integer", "description": "Transactions per second to generate"},
                                                        },
                                                },
                                        },
                                },
                        },
                        "responses": map[string]interface{}{
                                "200": map[string]interface{}{
                                        "description": "Comparison test started successfully",
                                },
                        },
                },
        }

        // Academic Testing endpoints
        paths["/api/v1/testing/benchmark/comprehensive"] = map[string]interface{}{
                "post": map[string]interface{}{
                        "tags":        []string{"Research & Testing"},
                        "summary":     "Run Comprehensive Benchmark",
                        "description": "Execute comprehensive performance benchmarks across all consensus algorithms with statistical analysis",
                        "responses": map[string]interface{}{
                                "200": map[string]interface{}{
                                        "description": "Benchmark started successfully",
                                },
                        },
                },
        }

        paths["/api/v1/testing/byzantine/launch-attack"] = map[string]interface{}{
                "post": map[string]interface{}{
                        "tags":        []string{"Research & Testing"},
                        "summary":     "Launch Byzantine Attack Simulation",
                        "description": "Launch a Byzantine fault injection attack for security testing",
                        "requestBody": map[string]interface{}{
                                "required": true,
                                "content": map[string]interface{}{
                                        "application/json": map[string]interface{}{
                                                "schema": map[string]interface{}{
                                                        "type": "object",
                                                        "properties": map[string]interface{}{
                                                                "attack_type": map[string]interface{}{"type": "string", "enum": []string{"double_spending", "fork_attack", "dos_attack", "selfish_mining", "nothing_at_stake", "eclipse_attack"}},
                                                                "intensity":   map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 10},
                                                                "duration":    map[string]interface{}{"type": "integer", "description": "Attack duration in seconds"},
                                                        },
                                                },
                                        },
                                },
                        },
                        "responses": map[string]interface{}{
                                "200": map[string]interface{}{
                                        "description": "Byzantine attack simulation started",
                                },
                        },
                },
        }

        return paths
}

// generateSwaggerComponents generates the OpenAPI components specification
func generateSwaggerComponents() SwaggerComponents {
        schemas := map[string]interface{}{
                "BlockchainInfo": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                                "chain_height":       map[string]interface{}{"type": "integer", "description": "Current blockchain height"},
                                "total_transactions": map[string]interface{}{"type": "integer", "description": "Total number of transactions"},
                                "consensus_algorithm": map[string]interface{}{"type": "string", "description": "Current consensus algorithm"},
                                "network_id":         map[string]interface{}{"type": "string", "description": "Network identifier"},
                                "genesis_hash":       map[string]interface{}{"type": "string", "description": "Genesis block hash"},
                                "latest_block_hash":  map[string]interface{}{"type": "string", "description": "Latest block hash"},
                                "timestamp":          map[string]interface{}{"type": "string", "format": "date-time"},
                        },
                },
                "Block": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                                "height":        map[string]interface{}{"type": "integer"},
                                "hash":          map[string]interface{}{"type": "string"},
                                "previous_hash": map[string]interface{}{"type": "string"},
                                "timestamp":     map[string]interface{}{"type": "string", "format": "date-time"},
                                "transactions":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"$ref": "#/components/schemas/Transaction"}},
                                "merkle_root":   map[string]interface{}{"type": "string"},
                                "difficulty":    map[string]interface{}{"type": "integer"},
                                "nonce":         map[string]interface{}{"type": "integer"},
                        },
                },
                "Transaction": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                                "hash":      map[string]interface{}{"type": "string"},
                                "from":      map[string]interface{}{"type": "string"},
                                "to":        map[string]interface{}{"type": "string"},
                                "amount":    map[string]interface{}{"type": "number"},
                                "fee":       map[string]interface{}{"type": "number"},
                                "timestamp": map[string]interface{}{"type": "string", "format": "date-time"},
                                "status":    map[string]interface{}{"type": "string", "enum": []string{"pending", "confirmed", "failed"}},
                                "block_height": map[string]interface{}{"type": "integer"},
                        },
                },
                "TransactionRequest": map[string]interface{}{
                        "type": "object",
                        "required": []string{"from", "to", "amount"},
                        "properties": map[string]interface{}{
                                "from":   map[string]interface{}{"type": "string", "description": "Sender address"},
                                "to":     map[string]interface{}{"type": "string", "description": "Recipient address"},
                                "amount": map[string]interface{}{"type": "number", "description": "Transaction amount"},
                                "fee":    map[string]interface{}{"type": "number", "description": "Transaction fee (optional)"},
                                "data":   map[string]interface{}{"type": "string", "description": "Additional transaction data (optional)"},
                        },
                },
                "TransactionResponse": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                                "hash":      map[string]interface{}{"type": "string", "description": "Transaction hash"},
                                "status":    map[string]interface{}{"type": "string", "description": "Transaction status"},
                                "timestamp": map[string]interface{}{"type": "string", "format": "date-time"},
                        },
                },
                "ConsensusStatus": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                                "algorithm":     map[string]interface{}{"type": "string", "description": "Current consensus algorithm"},
                                "current_round": map[string]interface{}{"type": "integer", "description": "Current consensus round"},
                                "view":          map[string]interface{}{"type": "integer", "description": "Current view number"},
                                "is_leader":     map[string]interface{}{"type": "boolean", "description": "Whether this node is the current leader"},
                                "performance_metrics": map[string]interface{}{
                                        "type": "object",
                                        "properties": map[string]interface{}{
                                                "tps":              map[string]interface{}{"type": "number", "description": "Transactions per second"},
                                                "latency_ms":       map[string]interface{}{"type": "number", "description": "Average latency in milliseconds"},
                                                "finality_time_ms": map[string]interface{}{"type": "number", "description": "Finality time in milliseconds"},
                                        },
                                },
                        },
                },
                "NetworkPeers": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                                "peer_count": map[string]interface{}{"type": "integer", "description": "Total number of connected peers"},
                                "peers": map[string]interface{}{
                                        "type": "array",
                                        "items": map[string]interface{}{
                                                "type": "object",
                                                "properties": map[string]interface{}{
                                                        "id":          map[string]interface{}{"type": "string"},
                                                        "address":     map[string]interface{}{"type": "string"},
                                                        "port":        map[string]interface{}{"type": "integer"},
                                                        "connected":   map[string]interface{}{"type": "boolean"},
                                                        "latency_ms":  map[string]interface{}{"type": "number"},
                                                        "last_seen":   map[string]interface{}{"type": "string", "format": "date-time"},
                                                        "consensus_algorithm": map[string]interface{}{"type": "string"},
                                                },
                                        },
                                },
                        },
                },
                "Error": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                                "error":     map[string]interface{}{"type": "string", "description": "Error message"},
                                "code":      map[string]interface{}{"type": "integer", "description": "Error code"},
                                "timestamp": map[string]interface{}{"type": "string", "format": "date-time"},
                        },
                },
        }

        return SwaggerComponents{
                Schemas: schemas,
        }
}