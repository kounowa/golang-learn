type EventHandler interface {
	HandleStoreEvent(ctx context.Context, message *event.EventMessage)
}

// chain 的每一个节点都要实现自己的 handler
type Chain struct {
	eventHandler EventHandler
	chain        *Chain
}

func NewChain(eventHandler EventHandler) *Chain {
	return &Chain{
		eventHandler: eventHandler,
	}
}

func (ch *Chain) SetNextEventHandler(eventHandler EventHandler) *Chain {
	next := NewChain(eventHandler)
	ch.chain = next
	return next
}

// 获取下游的 chain 对象
func (ch *Chain) GetNextChain() *Chain {
	return ch.chain
}

// 执行该链条
func (ch *Chain) HandleMsg(ctx context.Context, message *event.EventMessage) {
	rootChain := ch
	for {
		if rootChain.eventHandler != nil {
			rootChain.eventHandler.HandleStoreEvent(ctx, message)
		}
		nextChain := rootChain.GetNextChain()
		if nextChain != nil {
			rootChain = nextChain
		} else {
			// 最后一个节点的 nextChain 应该为空(这个时候就执行到最后一个节点了)
			return
		}
	}
}
