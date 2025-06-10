package xmysql

type Sharder struct {
	Main     *Handler
	Sharding []*Handler
}

func NewSharder(shardingCapacity int) *Sharder {
	var result = new(Sharder)
	result.Sharding = make([]*Handler, shardingCapacity)

	return result
}

func (s *Sharder) GetHandler(hash int64) *Handler {
	if hash == 0 {
		return s.Main
	}

	return s.Sharding[hash%int64(len(s.Sharding))]
}
