package operations

import (
	"context"
	"errors"
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	basev1 "jiffykube.io/jiffykube/api/v1"
)

type IngressClassAnnotation struct {
	Key   string
	Value string
}

// IngressName returns a formatted name for based on the parameterized App
func IngressName(app *basev1.App) string {
	return fmt.Sprintf("%s-ingress", app.Name)
}

// UpsertIngress updates an Ingress if it exists or creates it
func UpsertIngress(ctx context.Context, cli client.Client, app *basev1.App) error {
	action := "update"
	var ingress networkingv1.Ingress
	if err := cli.Get(ctx, client.ObjectKey{Name: IngressName(app), Namespace: app.Namespace}, &ingress); err != nil {
		if apierrors.IsNotFound(err) {
			action = "create"
		} else {
			return err
		}
	}
	fmt.Println(action, " ingress")
	ingressDefaults(&ingress, app)

	// cloud-specific
	switch app.Spec.IngressClass {
	case "gce":
		ingressGCE(&ingress, app)
	}

	switch action {
	case "create":
		err := cli.Create(ctx, &ingress)
		if err != nil {
			return err
		}
	case "update":
		err := cli.Update(ctx, &ingress)
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported action")
	}
	return nil
}

func ingressDefaults(ingress *networkingv1.Ingress, app *basev1.App) {
	var paths []networkingv1.HTTPIngressPath
	pathType := networkingv1.PathType("Prefix") // TODO - make configurable
	for _, rule := range app.Spec.Rules {
		paths = append(paths, networkingv1.HTTPIngressPath{
			Path:     rule.Path,
			PathType: &pathType,
			Backend: networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: ServiceName(app),
					Port: networkingv1.ServiceBackendPort{
						Number: 80,
					},
				},
			},
		})
	}
	ingress.ObjectMeta.Name = IngressName(app)
	ingress.ObjectMeta.Namespace = app.Namespace
	ingress.ObjectMeta.Labels = map[string]string{
		"app": app.Name,
	}
	ingress.Spec.Rules = []networkingv1.IngressRule{{
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: paths,
			},
		}},
	}
	ingress.Spec.IngressClassName = strPtr("nginx")
	if ingress.ObjectMeta.Annotations == nil {
		ingress.ObjectMeta.Annotations = make(map[string]string)
	}
	ingress.ObjectMeta.Annotations["ingressclass.kubernetes.io/is-default-class"] = "true" // TODO - should this be universal?
}

func ingressGCE(ingress *networkingv1.Ingress, app *basev1.App) {
	ingress.Spec.IngressClassName = nil
	ingress.ObjectMeta.Annotations["kubernetes.io/ingress.class"] = "gce"

}

// DeleteIngress deletes a Ingress if it exists
func DeleteIngress(ctx context.Context, cli client.Client, app *basev1.App) error {
	var ingress networkingv1.Ingress
	if err := cli.Get(ctx, client.ObjectKey{Name: IngressName(app), Namespace: app.Namespace}, &ingress); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if ingress.Name != "" {
		if err := cli.Delete(ctx, &ingress); err != nil {
			return err
		}
	}
	return nil
}
