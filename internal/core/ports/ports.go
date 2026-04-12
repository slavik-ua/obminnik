package ports

type OutboxNotifier interface {
	Notify()
}
