package main

type Backend interface {
	GetPublicKeys(token string) (string, error)
	AddPublicKey(token string, key []byte, truncate bool) error
	SetHashTable(token string, table []string) error
	IsHashExists(token string, hash string) (bool, error)
	GetHash(token string, number int64) (string, error)
	IsRecentClient(identifier string) (bool, error)
	AddRecentClient(identifier string) error
	GetTableSize(token string) (int64, error)
	GetTokens(prefix string) ([]string, error)

	Init() error
}
