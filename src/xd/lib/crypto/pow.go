package crypto

type POW interface {
	VerifyWork([]byte) bool
}
