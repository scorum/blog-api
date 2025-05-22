package main

//go:generate mockgen -source=push/notifier.go -package=push -destination=push/notifier_mock.go Notifier
