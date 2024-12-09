package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	n := int64(1)
	for {
		select {
		case <-ctx.Done():
			close(ch)
			return
		case ch <- n:
			fn(n)
			n++
		}
	}
}

func Worker(in <-chan int64, out chan<- int64) {
	for n := range in {
		out <- n
		time.Sleep(1 * time.Millisecond)
	}
	close(out)
}

func main() {
	chIn := make(chan int64)

	// Создание контекста
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var inputSum int64   // сумма сгенерированных чисел
	var inputCount int64 // количество сгенерированных чисел
	var mu sync.Mutex

	// Генерируем числа, считая их количество и сумму
	go Generator(ctx, chIn, func(i int64) {
		mu.Lock()
		inputSum += i
		inputCount++
		mu.Unlock()
	})

	const NumOut = 5 // количество обрабатывающих горутин и каналов
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	amounts := make([]int64, NumOut)
	chOut := make(chan int64, NumOut)

	var wg sync.WaitGroup

	// Собираем числа из каналов outs
	for i, out := range outs {
		wg.Add(1)
		go func(i int, out chan int64) {
			defer wg.Done()
			for n := range out {
				chOut <- n
				amounts[i]++
			}
		}(i, out)
	}

	// Запускаем горутину для закрытия chOut после завершения всех горутин
	go func() {
		wg.Wait()
		close(chOut)
	}()

	var count int64 // количество чисел результирующего канала
	var sum int64   // сумма чисел результирующего канала

	// Читаем числа из результирующего канала
	for n := range chOut {
		count++
		sum += n
	}

	fmt.Println("Количество чисел", inputCount, count)
	fmt.Println("Сумма чисел", inputSum, sum)
	fmt.Println("Разбивка по каналам", amounts)

	// Проверка результатов
	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
	for _, v := range amounts {
		inputCount -= v
	}
	if inputCount != 0 {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}
}
