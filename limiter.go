package limiter

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"time"
)

var Limit uint8 = 7
var Minutes = 20
var File = ""

type Payload struct {
	Email string `json:"email"`
	IP    string `json:"ip"`
}
type Info struct {
	Locked   bool
	Locks    int
	Attempts uint8
	UnlockAt int64
	Email    string
	IP       string
}

type Limiter struct {
	getRes   chan Info
	register chan Payload
	get      chan Payload
	wrong    chan Payload
	unlock   chan Payload
	reduce   chan Payload
}

func New() *Limiter {
	limiter := new()
	vals := map[string]*Info{}
	bts, err := os.ReadFile(File)
	if err == nil {
		b := bytes.Buffer{}
		b.Write(bts)
		dec := gob.NewDecoder(&b)
		err = dec.Decode(&vals)
		if err != nil {
			fmt.Println("error loading limiter info", err)
			vals = map[string]*Info{}
		}
	}
	go func() {
		for {
			select {
			case l := <-limiter.register:

				id := l.Email + "/" + l.IP
				if _, ok := vals[id]; !ok {
					vals[id] = &Info{
						Locked: false,
						Email:  l.Email,
						IP:     l.IP,
					}
				}
				val := vals[id]
				val.Attempts++
				if val.Attempts >= Limit {
					val.Attempts = 0
					val.Locks++
					lock := time.Now()
					for i := 0; i < val.Locks; i++ {
						mins := Minutes * (i + 1) * (i + 1) * (i + 1)
						lock = lock.Add(time.Minute * time.Duration(mins))
					}
					val.Locked = true
					val.UnlockAt = lock.Unix()
				}
				err = Save(vals)
				if err != nil {
					fmt.Println("error saving file:", err)
				}
			case l := <-limiter.get:
				id := l.Email + "/" + l.IP
				if val, ok := vals[id]; ok {
					limiter.getRes <- *val
				} else {
					limiter.getRes <- Info{}
				}
			case l := <-limiter.unlock:
				id := l.Email + "/" + l.IP
				if val, ok := vals[id]; ok {
					val.Locked = false
					val.UnlockAt = 0
					val.Attempts = 0
				}
				err = Save(vals)
				if err != nil {
					fmt.Println("error saving file:", err)
				}
			case l := <-limiter.reduce:
				id := l.Email + "/" + l.IP
				if val, ok := vals[id]; ok {
					if val.Locks > 0 {
						val.Locks--
						if val.Locks == 0 {
							delete(vals, id)
						}
					}
				}
				err = Save(vals)
				if err != nil {
					fmt.Println("error saving file:", err)
				}

			}
		}
	}()

	return limiter
}

func (l *Limiter) Get(p Payload) Info {
	l.get <- p
	res := <-l.getRes
	return res
}
func (l *Limiter) Wrong(p Payload) {
	l.register <- p
}
func (l *Limiter) Unlock(p Payload) {
	l.unlock <- p
}
func (l *Limiter) Reduce(p Payload) {
	l.reduce <- p
}

func new() *Limiter {
	return &Limiter{
		getRes:   make(chan Info),
		get:      make(chan Payload),
		wrong:    make(chan Payload),
		unlock:   make(chan Payload),
		register: make(chan Payload),
		reduce:   make(chan Payload),
	}
}

func Save(v interface{}) error {
	if File == "" {
		return nil
	}
	b := bytes.Buffer{}
	enc := gob.NewEncoder(&b)
	err := enc.Encode(v)
	if err != nil {
		return err
	}
	err = os.WriteFile(File, b.Bytes(), 0644)
	if err != nil {
		return err
	}
	return nil
}

// Example
// func example(l *Limiter) fiber.Handler {
// 	return func(ctx *fiber.Ctx) error {
// 		info := Payload{}
// 		err := ctx.BodyParser(&info)
// 		if err != nil {
// 			return err
// 		}
// 		info.IP = ctx.IP()
// 		v := l.Get(info)
// 		if v.Locked {
// 			if v.UnlockAt <= time.Now().Unix() {
// 				l.Unlock(info)
// 			} else {
// 				return ctx.Status(fiber.StatusLocked).SendString(fmt.Sprintf("%d", v.UnlockAt))
// 			}
// 		}
// 		n := rand.Float64()
// 		//if password is correct or not
// 		if n >= 0.5 {
// 			l.Wrong(info)
// 			return ctx.Status(fiber.StatusUnauthorized).SendString(fmt.Sprintf("%d", Limit-v.Attempts))
// 		}
// 		l.Reduce(info)
// 		l.Unlock(info)
// 		return nil
// 	}
// }
var Example struct{}
