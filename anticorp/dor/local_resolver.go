package dor

import "context"

type localResolver struct {
	names map[string]string
}

func NewLocalResolver() Resolver {
	return &localResolver{
		names: map[string]string{},
	}
}

func (r *localResolver) Add(ctx context.Context, name, value string) error {
	_, er := getRecordFromName(name)
	if er != nil {
		return er
	}
	r.names[name] = value
	return nil
}

func (r *localResolver) Resolve(ctx context.Context, name string) (string, error) {
	rec, er := getRecordFromName(name)
	if er != nil {
		return "", er
	}
	res := r.names[rec.Query]
	return res, nil
}
