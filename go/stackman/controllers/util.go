package controllers

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

var httpRe *regexp.Regexp
var pathRe *regexp.Regexp
var portRe *regexp.Regexp

func Hostify(in string) string {
	port := 80
	if strings.HasPrefix(in, "https://") {
		port = 443
	}

	out := httpRe.ReplaceAllString(in, "")
	out = pathRe.ReplaceAllString(out, "")
	if portRe.MatchString(out) {
		return out
	}

	return out + ":" + strconv.Itoa(port)
}

func ObjectName(om v1.ObjectMeta, name string) string {
	return fmt.Sprintf("%s-%s", om.Name, name)
}

func ObjectNamespacedName(om v1.ObjectMeta, name string) types.NamespacedName {
	return types.NamespacedName{
		Name:      ObjectName(om, name),
		Namespace: om.Namespace,
	}
}

func ObjectMeta(om v1.ObjectMeta, name string, labels map[string]string) v1.ObjectMeta {
	return v1.ObjectMeta{
		Name:      ObjectName(om, name),
		Namespace: om.Namespace,
		Labels:    labels,
	}
}

type Creator func() client.Object

type ReaderWriter interface {
	client.Reader
	client.Writer
}

func GetOrCreateResource(
	ctx context.Context,
	client ReaderWriter,
	creator Creator,
	name types.NamespacedName,
	proto client.Object,
) (bool, error) {
	err := client.Get(ctx, name, proto)
	if err == nil {
		return false, nil
	}

	if errors.IsNotFound(err) {
		err = client.Create(ctx, creator())
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, err
}

func init() {
	httpRe = regexp.MustCompile("^http(s)?://")
	pathRe = regexp.MustCompile("/(.*)$")
	portRe = regexp.MustCompile(":([\\d]+)$")
}
