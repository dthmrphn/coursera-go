package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

func hash(in, out chan interface{}, cb func(string) string) {
	wg := sync.WaitGroup{}

	for i := range in {
		wg.Add(1)
		go func(str string) {
			defer wg.Done()
			out <- cb(str)
		}(fmt.Sprintf("%v", i))
	}

	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	mu := sync.Mutex{}

	hasher := func(data string) string {
		ch1 := make(chan string)
		ch2 := make(chan string)

		go func() {
			defer close(ch1)

			ch1 <- DataSignerCrc32(data)
		}()

		go func() {
			defer close(ch2)

			mu.Lock()
			h := DataSignerMd5(data)
			mu.Unlock()

			ch2 <- DataSignerCrc32(h)
		}()

		return fmt.Sprintf("%s~%s", <-ch1, <-ch2)
	}

	hash(in, out, hasher)
}

func MultiHash(in, out chan interface{}) {
	hasher := func(data string) string {
		hash := make([]string, 6)
		wg := sync.WaitGroup{}

		for i := range hash {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				hash[i] = DataSignerCrc32(fmt.Sprintf("%d%s", i, data))
			}(i)
		}

		wg.Wait()

		return strings.Join(hash, "")

	}

	hash(in, out, hasher)
}

func CombineResults(in, out chan interface{}) {
	data := make([]string, 0)

	for i := range in {
		data = append(data, fmt.Sprint(i))
	}

	sort.Strings(data)
	out <- fmt.Sprint(strings.Join(data, "_"))
}

func ExecutePipeline(jobs ...job) {
	chs := make([]chan interface{}, len(jobs)+1)
	for i := 0; i < len(jobs)+1; i++ {
		chs[i] = make(chan interface{})
	}

	wg := sync.WaitGroup{}
	for i := range jobs {
		wg.Add(1)
		go func(i int, j job, in, out chan interface{}) {
			j(in, out)
			wg.Done()
			close(out)
		}(i, jobs[i], chs[i], chs[i+1])
	}
	wg.Wait()
}

