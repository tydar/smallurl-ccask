package main

import (
	"errors"
	"fmt"

	"github.com/tydar/smallurl-ccask/ccask"
)

var ErrNoSuchKey = errors.New("no such key")

type ShortLinkRepository interface {
	GetLink(key string) (ShortLink, error)
	SetLink(key, url string) error
}

type ShortLinkModel struct {
	client *ccask.CCaskClient
}

func NewShortLinkModel(client *ccask.CCaskClient) *ShortLinkModel {
	return &ShortLinkModel{client}
}

type ShortLink struct {
	Key string
	URL string
}

func (slm *ShortLinkModel) GetLink(key string) (ShortLink, error) {
	res, err := slm.client.GetRes([]byte(key))
	if err != nil {
		return ShortLink{}, fmt.Errorf("client.GetRes: %v", err)
	}

	switch res.ResCode() {
	case ccask.GET_SUCCESS:
		retStr := string(res.Value())
		return ShortLink{Key: key, URL: retStr}, nil
	case ccask.GET_FAIL:
		return ShortLink{}, fmt.Errorf("%w", ErrNoSuchKey)
	default:
		return ShortLink{}, fmt.Errorf("Invalid response type from database: %v\n", res.ResCode())
	}
}

func (slm *ShortLinkModel) SetLink(key, url string) error {
	res, err := slm.client.SetRes([]byte(key), []byte(url))
	if err != nil {
		return fmt.Errorf("client.SetRes: %v", err)
	}

	switch res.ResCode() {
	case ccask.SET_FAIL:
		return fmt.Errorf("SET request failed; please try again.")
	case ccask.SET_SUCCESS:
		return nil
	default:
		return fmt.Errorf("Unexpected response code %d; please try again.", res.ResCode())
	}
}
