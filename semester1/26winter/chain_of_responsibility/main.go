package main

import (
	"fmt"
	"math/rand"
)

type Part interface {
	execute(*Customer)
	setNext(Part)
}

type Waiter struct {
	next Part
}

func (w *Waiter) execute(c *Customer) {
	fmt.Println("Start to place an order")

	if !c.orderPlaced {
		if rand.Float32() >= 0.4 {
			fmt.Println("Oops, waiter can not recognize the customer's accent!")
			return
		} else {
			fmt.Println("[Order] placed")
			c.orderPlaced = true
			w.next.execute(c)
			return
		}
	} else {
		fmt.Println("[Order] already placed")
		w.next.execute(c)
	}
}

func (w *Waiter) setNext(next Part) {
	w.next = next
}

type Frontend struct {
	next Part
}

func (f *Frontend) execute(c *Customer) {
	if !c.orderTransed {
		if rand.Float32() >= 0.5 {
			fmt.Println("Oops, frontend failed")
			return
		} else {
			fmt.Println("[Order] transported")
			c.orderTransed = true
			f.next.execute(c)
			return
		}
	} else {
		fmt.Println("[Order] already transported")
		f.next.execute(c)
	}
}

func (f *Frontend) setNext(next Part) {
	f.next = next
}

type Chef struct {
	next Part
}

func (cf *Chef) execute(c *Customer) {
	if !c.orderDone {
		if rand.Float32() >= 0.9 {
			fmt.Println("Oops, the chef messed it up!")
			return
		} else {
			fmt.Println("[Order] Done")
			c.orderDone = true
			return
		}
	} else {
		fmt.Println("[order] already done")
	}
}

func (cf *Chef) setNext(next Part) {
	cf.next = next
}

type Broadcast struct {
	next Part
}

func (b *Broadcast) execute(c *Customer) {
	if !c.orderDone {
		fmt.Println("Order for ", c.name, " now processing")
		return
	} else {
		fmt.Println("Order is completed for customer: ", c.name)
	}
}

func (b *Broadcast) setNext(next Part) {
	b.next = nil
}

type Customer struct {
	name         string
	orderPlaced  bool
	orderTransed bool
	orderDone    bool
}

func main() {
	Broadcast := &Broadcast{}
	Broadcast.setNext(nil)

	Chef := &Chef{}
	Chef.setNext(Broadcast)

	Frontend := &Frontend{}
	Frontend.setNext(Chef)

	Waiter := &Waiter{}
	Waiter.setNext(Frontend)

	Customer := &Customer{name: "Bob"}

	for range 10 {
		Waiter.execute(Customer)
		Customer.orderPlaced = false
	}
}
