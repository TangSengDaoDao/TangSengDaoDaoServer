package wkrsa

import (
	"fmt"
	"testing"
)

func TestSignWithMD5(t *testing.T) {

	dz := `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAxGbKnrOumh0r4FfRDL3NIGP7scxxvsVupktTdFqf8TgHLT67
qfMXxM/JSAmNxJVz4zS8zVla4FlTJ4kyh3IJW+65vzXKX/errnI0gNk0p3IME+Wd
LDetwy04Dk40B4GhMrfnomMff6W7UXG4fyLX9gzCIhkAeLNNl6eZS8IOgFULWZRO
pYdlylncY4EQiv6ljXQtKmTIxvi7ivS/9sSFqgS+qnbd0QexrZlM1D2O4abFgrjs
BaFHC4zGrg3TztiWynGUSOpjFv2v92doxDgSxP69xNry9Vq+6kERJpbqvJ3ujUEY
T4bLk8avXNjyMg79/xNeiQ4gxvSHj3v3YOOlIQIDAQABAoIBADbY9fDIARSs3Nnz
7D+Aqc5H3bxTedhqznHGS3IM9OmqWea6xDG734Fo/a8Oa/bgPdLPoYI/V++bQmui
FuhYYmC4FEtfvDp8sgcvgZYSEnBImzLbRr9YdUAyWps0H7eQ7fF6BkgFIoDFScB+
36Uxl9nwyi43iTgr6plVhqvvb5lKqPQTZWWI50hs0EYcADngy2st6+sJxJXylCI4
w3vjXhE9WdwBk54QD6FPPrzRHWpqHDjPcuyxbVijEoz0YMdO+tnHweUn+YnQnHxi
rswF7b9/DL8mhIX67SKqF7wM+RCgrweVswNCcEsZyfCtMKgI7uwDjaDkvhI+XaYq
vfNUe9kCgYEAzK+Zb3A4ZxPWhYrubqmGYa7KvmtetRO5LAyWXMFH5vzU+TKzDcJm
qZTxhzH8yt43PgrjOL5C+X2cIkNfEXHl9oIuuAMbMDSqThvox+o8Lsz5gTKy9/rR
sucBRc41N/axLalVpbevD3Q3+2WjH25ap7h0RsZsladshEoxXI8jpCMCgYEA9aOD
SSoywbVrsMXhr2wgxNXUuQGxWH6x3ZbAG86xxerElVwT06SbowDO77xFmiQsUvlW
BpYrVPBg48B/Sgvvb0mKHhXVe8Maegq6bCofVpAU9IhDZokwIGJgEm2MgziZJAe+
3/2DCFV22olrkqDhllqBTenuSyiIfWcGHr+zM+sCgYB0EWNVgPJK6UHtajH4iKMO
Q1rujd4fmnaXlu+w211Vi6uNQAWu2Lz0juRDQMJTm50BzpS4uZMq/OKLv15qewbn
OT0a1ZAWTtcAAe2HZ7kG5O7bJ4+69PzykPH0zpD5Eie4d9x8Y2OexM11/lV43lAD
6aHt/FjYqB7uCVBiZzzTtwKBgQDGY+jN9+IEp4UxwbCUYQ1aTKXBQoe8xJ7dLDs+
ekMEaaeaRkLRJdp53VZFM9c3Nk4COdTr/u9Ca96lM7zazib0x/1gbRv+GEbTGMUW
RTMIU9hI46EkOFsBXNLhL09UUCsHeaYE/JiO64/R0zlptLxeFfznM6+9TiBmwAWm
YgfXPwKBgDn1n2aWpGpH/cz8AQapMxqo7YK2Lc/Vwqw7O4mmNQ0TOJOgECHrWJX2
2UfOnyh8mDNFq9Udt4XMkfH5aPmWF7ejl2ddz3A4kdz5DMaVgRPktJh6nFAllKz/
R0a+FMM2fsqqRjYN4Zf84pJvWnIy7dG/pXKq8AAWkV7iJNVpshGb
-----END RSA PRIVATE KEY-----`
	fmt.Println("privateKeyBuff.Bytes()-->", dz)

	_, err := SignWithMD5([]byte("test"), []byte(dz))
	if err != nil {
		panic(err)
	}
}
