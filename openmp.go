package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type NodosMenorLatencia struct {
	nodos   []int
	cerrojo sync.Mutex
}

func EnviarPing(latencia int, out chan<- string) {
	time.Sleep(time.Duration(latencia) * time.Millisecond)
	out <- fmt.Sprintf("Ping con %d ms", latencia)
}

func (n *NodosMenorLatencia) GuardarNodoMenorLatencia(latencias map[int]int) {
	n.cerrojo.Lock()
	defer n.cerrojo.Unlock()

	menorLatencia := 99999999
	var idNodo int
	for id, latencia := range latencias {
		if latencia < menorLatencia {
			menorLatencia = latencia
			idNodo = id
		}
	}

	//fmt.Println(latencias)

	n.nodos = append(n.nodos, idNodo+1)
}

func oepnmp() {
	var wg sync.WaitGroup
	done := make(chan struct{})

	canales := make([]chan string, 3)
	for i := 0; i < 3; i++ {
		canales[i] = make(chan string, 1)
	}

	n := NodosMenorLatencia{}

	wg.Add(3)
	for i := 0; i < 3; i++ {
		go func(id int) {
			for {
				select {
				case <-done:
					fmt.Printf("Finalizó el nodo %d\n", id+1)
					wg.Done()
					return
				case msj := <-canales[id]:
					fmt.Printf("Nodo %d recibió: %s\n", id+1, msj)
				}
			}
		}(i)
	}

	wg.Add(1)
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(2 * time.Second)

			latencias := make(map[int]int, 3)
			for j := 0; j < 3; j++ {
				latencias[j] = rand.Intn(401) + 100
				EnviarPing(latencias[j], canales[j])
			}

			n.GuardarNodoMenorLatencia(latencias)
		}
		close(done)
		wg.Done()
	}()

	wg.Wait()
	fmt.Println("Terminó el tiempo")

	fmt.Println("Nodos con menor latencia:", n.nodos)
}
