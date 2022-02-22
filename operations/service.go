package operations

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	basev1 "jiffykube.io/jiffykube/api/v1"
)

// ServiceName returns a formatted name for based on the parameterized App
func ServiceName(app *basev1.App) string {
	return fmt.Sprintf("%s-service", app.Name)
}

// UpsertService gets a Deployment and Updates it or Creates it if it does not exist
func UpsertService(ctx context.Context, cli client.Client, app *basev1.App) error {

	action := "update"
	var service corev1.Service
	if err := cli.Get(ctx, client.ObjectKey{Name: ServiceName(app), Namespace: app.Namespace}, &service); err != nil {
		if apierrors.IsNotFound(err) {
			action = "create"
		} else {
			return err
		}
	}
	fmt.Println(action, " service")
	serviceDefaults(&service, app)

	// Cloud-specific values
	switch app.Spec.IngressClass {
	case "gce":
		serviceGCE(&service, app)
	}

	switch action {
	case "create":
		err := cli.Create(ctx, &service)
		if err != nil {
			return err
		}
	case "update":
		err := cli.Update(ctx, &service)
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported action")
	}
	return nil
}

func serviceDefaults(service *corev1.Service, app *basev1.App) {
	var containers []corev1.Container
	for _, container := range app.Spec.Containers {
		containers = append(containers, corev1.Container{
			Ports: []corev1.ContainerPort{
				{
					Name:          fmt.Sprintf("%s-app-port", app.Name),
					ContainerPort: int32(container.Ports.ContainerPort),
				},
			},
			Name:  container.Name,
			Image: container.Image,
		})
	}
	service.ObjectMeta.Name = ServiceName(app)
	service.ObjectMeta.Namespace = app.Namespace
	service.ObjectMeta.Labels = map[string]string{
		"app": app.Name,
	}
	service.Spec.Type = corev1.ServiceTypeNodePort
	service.Spec.Selector = map[string]string{
		"app": app.Name,
	}
	service.Spec.Ports = []corev1.ServicePort{
		{
			Name:       fmt.Sprintf("%s-app-port", app.Name),
			Port:       80,
			TargetPort: intstr.FromInt(app.Spec.Containers[0].Ports.ContainerPort), // TODO - check index?
			Protocol:   "TCP",
		},
	}
}

func serviceGCE(service *corev1.Service, app *basev1.App) {
	if service.Annotations == nil {
		service.Annotations = make(map[string]string)
	}
	service.Annotations["cloud.google.com/neg"] = `{"ingress": true}`
}

// DeleteService deletes a Service if it exists
func DeleteService(ctx context.Context, cli client.Client, app *basev1.App) error {
	var service corev1.Service
	if err := cli.Get(ctx, client.ObjectKey{Name: ServiceName(app), Namespace: app.Namespace}, &service); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if service.Name != "" {
		if err := cli.Delete(ctx, &service); err != nil {
			return err
		}
	}
	return nil
}
