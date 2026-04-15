package planner

import (
"math/rand"
"time"
)

func init() {
rand.Seed(time.Now().UnixNano())
}

func randomString(n int) string {
letters := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
b := make([]rune, n)
for i := range b {
b[i] = letters[rand.Intn(len(letters))]
}
return string(b)
}
