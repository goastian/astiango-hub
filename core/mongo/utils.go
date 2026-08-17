package mongo

import "go.mongodb.org/mongo-driver/bson"

func GetMongoQuery(query ListQuery) (res bson.M) {
	res = bson.M{}
	for _, c := range query {
		switch c.Op {
		case OpEqual:
			res[c.Key] = c.Value
		default:
			res[c.Key] = bson.M{
				c.Op: c.Value,
			}
		}
	}
	return res
}

func GetMongoOpts(opts *ListOptions) (res *FindOptions) {
	var sort bson.D
	for _, s := range opts.Sort {
		direction := 1
		if s.Direction == SortDirectionAsc {
			direction = 1
		} else if s.Direction == SortDirectionDesc {
			direction = -1
		}
		sort = append(sort, bson.E{Key: s.Key, Value: direction})
	}
	return &FindOptions{
		Skip:  opts.Skip,
		Limit: opts.Limit,
		Sort:  sort,
	}
}
