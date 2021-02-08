type Factory interface {
	CreateWorkerFunc(namespace string, taskType string, id string) func(ctx context.Context) error
}

type WorkerLifeManager struct {
	recorders record.Factory
	table     map[string]*Worker
	rw        sync.RWMutex
	factory   Factory
}

// 生命周期管理(控制多个协程的生命周期
func NewLifeManager(factory Factory, recorders record.Factory) *WorkerLifeManager {
	p := &WorkerLifeManager{
		factory:   factory,
		table:     map[string]*Worker{},
		recorders: recorders,
	}
	go p.loop()
	return p
}

// 开始
func (lives *WorkerLifeManager) Start(ctx context.Context, namespace string, taskType string, id string) {
	tableKey := fmt.Sprintf("%s-%s-%s", namespace, taskType, id)
	easyrecord.DoWithRecords(ctx, func(ctx context.Context) error {
		lives.rw.RLock()
		v, ok := lives.table[tableKey]
		lives.rw.RUnlock()
		if !ok || v == nil {
			lives.rw.Lock()
			defer lives.rw.Unlock()
			v, ok = lives.table[tableKey]
			if !ok || v == nil {
				p := lives.newProcessor(namespace, taskType, id)
				// 构造 processor 不成功返回（可以用于先判断比赛是否开始 没开始则返回,不处理）
				if p == nil {
					return nil
				}
				lives.table[tableKey] = p
			}
		}
		if v != nil {
			v.deadLine = time.Now().Add(time.Hour)
		}
		return nil
	}, lives.recorders, "WorkerLifeManager.Start", record.StringField("tableKey", tableKey))
}

// 强制使得协程停止
func (lives *WorkerLifeManager) ForceStop(ctx context.Context, namespace string, taskType string, id string) {
	key := fmt.Sprintf("%s-%s-%s", namespace, taskType, id)
	lives.rw.Lock()
	p := lives.table[key]
	defer lives.rw.RUnlock()
	if p != nil {
		delete(lives.table, key)
		p.Close()
		log.Println("WorkerLifeManager ForceStop process", key, p.deadLine)
	}
}

func (lives *WorkerLifeManager) newProcessor(namespace string, taskType string, id string) *Worker {
	log.Println("WorkerLifeManager start to run", namespace, taskType, id)
	ctx, cancel := context.WithCancel(context.Background())
	// 这一步会有点阻塞 启动处理器的预处理
	f := lives.factory.CreateWorkerFunc(namespace, taskType, id)
	// 构造处理函数不成功 返回nil
	if f == nil {
		log.Println("WorkerLifeManager new processor return nil", namespace, taskType, id)
		return nil
	}
	p := &Worker{
		// 这边也需要添加超时时间
		deadLine: time.Now().Add(time.Hour),
		ctx:      ctx,
		cancel:   cancel,
		run:      f,
	}
	p.StartRun()
	log.Println("WorkerLifeManager end to run", namespace, taskType, id)
	return p
}

func (lives *WorkerLifeManager) loop() {
	var keys = []string{}
	for range time.NewTicker(time.Minute).C {
		keys = keys[:0]
		lives.rw.RLock()
		for key := range lives.table {
			keys = append(keys, key)
		}
		lives.rw.RUnlock()
		for _, key := range keys {
			lives.rw.RLock()
			p := lives.table[key]
			lives.rw.RUnlock()
			if p != nil && p.deadLine.Before(time.Now()) {
				lives.rw.Lock()
				delete(lives.table, key)
				lives.rw.Unlock()
				p.Close()
				log.Println("WorkerLifeManager close process", key, p.deadLine)
			}
		}
	}
}

type Worker struct {
	deadLine time.Time
	cancel   func()
	ctx      context.Context
	run      func(ctx context.Context) error
}

func (p *Worker) StartRun() {
	if p.run == nil {
		return
	}
	go p.run(p.ctx)
}

func (p *Worker) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}
