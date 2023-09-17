package bus

const (
	TopicBlockFound = "block.found"
)

func init() {
	InitBus()

	Bus.RegisterTopics(
		TopicBlockFound,
	)
}
