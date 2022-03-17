package Shared

import (
	"os"
	"sync"
)

var (
	CManager      = &ComboManager{}
	PManager      = &ProxyManager{}
	BotCount  int = 250
	WaitGroup     = sync.WaitGroup{}
	Semaphore chan int

	HitChan chan Account
	OutFile *os.File
)
