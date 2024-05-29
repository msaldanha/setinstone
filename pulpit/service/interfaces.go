package service

import "github.com/msaldanha/setinstone/pulpit/models"

type Bolt interface {
}
type SubscriptionsStore interface {
	AddSubscription(subscription models.Subscription) error
	RemoveSubscription(subscription models.Subscription) error
	GetAllSubscriptions(ns string, address string) ([]models.Subscription, error)
}
