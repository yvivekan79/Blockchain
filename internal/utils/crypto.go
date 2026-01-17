package utils

import (
        "crypto/ecdsa"
        "crypto/elliptic"
        "crypto/rand"
        "crypto/sha256"
        "encoding/hex"
        "errors"
        "fmt"
        "math/big"

        "golang.org/x/crypto/ripemd160"
)

// GenerateKeyPair generates a new ECDSA key pair
func GenerateKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
        curve := elliptic.P256()
        privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
        if err != nil {
                return nil, nil, fmt.Errorf("failed to generate key pair: %w", err)
        }
        
        return privateKey, &privateKey.PublicKey, nil
}

// PublicKeyToAddress converts a public key to a blockchain address
func PublicKeyToAddress(pubKey *ecdsa.PublicKey) string {
        // Serialize public key
        pubKeyBytes := append(pubKey.X.Bytes(), pubKey.Y.Bytes()...)
        
        // SHA256 hash
        sha256Hash := sha256.Sum256(pubKeyBytes)
        
        // RIPEMD160 hash
        ripemd160Hasher := ripemd160.New()
        ripemd160Hasher.Write(sha256Hash[:])
        hash := ripemd160Hasher.Sum(nil)
        
        // Add version byte (0x00 for mainnet)
        versionedHash := append([]byte{0x00}, hash...)
        
        // Double SHA256 for checksum
        checksum1 := sha256.Sum256(versionedHash)
        checksum2 := sha256.Sum256(checksum1[:])
        
        // Add first 4 bytes of checksum
        fullHash := append(versionedHash, checksum2[:4]...)
        
        return hex.EncodeToString(fullHash)
}

// Sign signs data with a private key
func Sign(privateKey *ecdsa.PrivateKey, data []byte) (string, error) {
        hash := sha256.Sum256(data)
        
        r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
        if err != nil {
                return "", fmt.Errorf("failed to sign data: %w", err)
        }
        
        // Encode signature
        signature := append(r.Bytes(), s.Bytes()...)
        return hex.EncodeToString(signature), nil
}

// Verify verifies a signature against data and public key
func Verify(publicKey *ecdsa.PublicKey, data []byte, signature string) (bool, error) {
        sigBytes, err := hex.DecodeString(signature)
        if err != nil {
                return false, fmt.Errorf("failed to decode signature: %w", err)
        }
        
        if len(sigBytes) != 64 {
                return false, errors.New("invalid signature length")
        }
        
        r := big.NewInt(0).SetBytes(sigBytes[:32])
        s := big.NewInt(0).SetBytes(sigBytes[32:])
        
        hash := sha256.Sum256(data)
        
        return ecdsa.Verify(publicKey, hash[:], r, s), nil
}

// Hash calculates SHA256 hash of data
func Hash(data []byte) string {
        hash := sha256.Sum256(data)
        return hex.EncodeToString(hash[:])
}

// HashString calculates SHA256 hash of string
func HashString(data string) string {
        return Hash([]byte(data))
}

// DoubleHash calculates double SHA256 hash  
func DoubleHash(data []byte) string {
        first := sha256.Sum256(data)
        second := sha256.Sum256(first[:])
        return hex.EncodeToString(second[:])
}



// GenerateRandomString generates a random hex string
func GenerateRandomString(length int) (string, error) {
        bytes := make([]byte, length/2)
        if _, err := rand.Read(bytes); err != nil {
                return "", fmt.Errorf("failed to generate random string: %w", err)
        }
        return hex.EncodeToString(bytes), nil
}

// MerkleRoot calculates the merkle root of a list of hashes
func MerkleRoot(hashes []string) string {
        if len(hashes) == 0 {
                return ""
        }
        
        if len(hashes) == 1 {
                return hashes[0]
        }
        
        var newHashes []string
        
        for i := 0; i < len(hashes); i += 2 {
                var combined string
                if i+1 < len(hashes) {
                        combined = hashes[i] + hashes[i+1]
                } else {
                        combined = hashes[i] + hashes[i]
                }
                
                hash := sha256.Sum256([]byte(combined))
                newHashes = append(newHashes, hex.EncodeToString(hash[:]))
        }
        
        return MerkleRoot(newHashes)
}

// ValidateAddress validates a blockchain address format
func ValidateAddress(address string) bool {
        if len(address) == 0 {
                return false
        }
        
        // Accept Ethereum-style addresses (0x + 40 hex chars = 42 total)
        if len(address) == 42 && address[:2] == "0x" {
                // Check if remaining 40 characters are valid hex
                _, err := hex.DecodeString(address[2:])
                return err == nil
        }
        
        // Original validation for internal format (50 hex characters)
        if len(address) == 50 {
                addrBytes, err := hex.DecodeString(address)
                if err != nil {
                        return false
                }
                
                if len(addrBytes) != 25 {
                        return false
                }
                
                // Extract checksum
                payload := addrBytes[:21]
                checksum := addrBytes[21:]
                
                // Verify checksum
                hash1 := sha256.Sum256(payload)
                hash2 := sha256.Sum256(hash1[:])
                
                return hex.EncodeToString(checksum) == hex.EncodeToString(hash2[:4])
        }
        
        return false
}

// GenerateNonce generates a secure random nonce
func GenerateNonce() (int64, error) {
        max := big.NewInt(1000000000) // 1 billion
        n, err := rand.Int(rand.Reader, max)
        if err != nil {
                return 0, fmt.Errorf("failed to generate nonce: %w", err)
        }
        return n.Int64(), nil
}

// CalculateHash calculates SHA256 hash of string data
func CalculateHash(data string) string {
        return HashString(data)
}

// HashDifficulty checks if hash meets difficulty requirement
func HashDifficulty(hash string, difficulty int) bool {
        if len(hash) < difficulty {
                return false
        }
        
        for i := 0; i < difficulty; i++ {
                if hash[i] != '0' {
                        return false
                }
        }
        
        return true
}

// GenerateShardKey generates a deterministic shard key from address
func GenerateShardKey(address string, numShards int) int {
        hash := sha256.Sum256([]byte(address))
        key := big.NewInt(0).SetBytes(hash[:])
        mod := big.NewInt(int64(numShards))
        result := big.NewInt(0).Mod(key, mod)
        return int(result.Int64())
}

// EncryptData encrypts data using AES (placeholder for actual implementation)
func EncryptData(data []byte, key []byte) ([]byte, error) {
        // This is a placeholder - implement actual AES encryption
        // For now, just return the data (not secure)
        return data, nil
}

// DecryptData decrypts data using AES (placeholder for actual implementation)
func DecryptData(encryptedData []byte, key []byte) ([]byte, error) {
        // This is a placeholder - implement actual AES decryption
        // For now, just return the data (not secure)
        return encryptedData, nil
}
