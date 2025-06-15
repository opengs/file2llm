package parser

/*
import "context"

type StreamResultIteratorProxy struct {
	base        StreamResultIterator
	nextFunc    func(ctx context.Context, base StreamResultIterator) bool
	currentFunc func(base StreamResultIterator) StreamResult
	closeFunc   func(base StreamResultIterator)
}

func (i *StreamResultIteratorProxy) Next(ctx context.Context) bool {
	if i.nextFunc != nil {
		return i.nextFunc(ctx, i.base)
	}

	return i.base.Next(ctx)
}

func (i *StreamResultIteratorProxy) Current() StreamResult {
	if i.currentFunc != nil {
		return i.currentFunc(i.base)
	}

	return i.base.Current()
}

func (i *StreamResultIteratorProxy) Close() {
	if i.closeFunc != nil {
		i.closeFunc(i.base)
		return
	}

	i.base.Close()
}
*/
