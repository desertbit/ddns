//go:generate msgp

package db

type Record struct {
	RR      string `msg:"rr"`
	Expires int64  `msg:"expires"` // Unix timestamp. Never expires if <= 0
}
