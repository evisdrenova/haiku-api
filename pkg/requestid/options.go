package requestid

type options struct {
	gen IDGenerator
}

type Option func(*options)

type IDGenerator func() string

func WithIDGenerator(gen IDGenerator) Option {
	return func(opt *options) {
		opt.gen = gen
	}
}
