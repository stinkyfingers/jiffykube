package operations

import (
	"context"
	"errors"
	"fmt"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	basev1 "jiffykube.io/jiffykube/api/v1"
)

// DeploymentName returns a formatted name for based on the parameterized App
func DeploymentName(app *basev1.App) string {
	return fmt.Sprintf("%s-deployment", app.Name)
}

// UpsertDeployent gets a Deployment and Updates it or Creates it if it does not exist
func UpsertDeployment(ctx context.Context, cli client.Client, app *basev1.App) error {

	action := "update"
	var deployment v1.Deployment
	if err := cli.Get(ctx, client.ObjectKey{Name: DeploymentName(app), Namespace: app.Namespace}, &deployment); err != nil {
		if apierrors.IsNotFound(err) {
			action = "create"
		} else {
			return err
		}
	}
	fmt.Println(action, " deployment")
	deploymentDefaults(&deployment, app)

	switch action {
	case "create":
		err := cli.Create(ctx, &deployment)
		if err != nil {
			return err
		}
	case "update":
		err := cli.Update(ctx, &deployment)
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported action")
	}
	return nil
}

func deploymentDefaults(deployment *v1.Deployment, app *basev1.App) {
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
	deployment.ObjectMeta.Name = DeploymentName(app)
	deployment.ObjectMeta.Namespace = app.Namespace
	deployment.ObjectMeta.Labels = map[string]string{
		"app": app.Name,
	}
	deployment.Spec.Replicas = int32Ptr(app.Spec.Replicas)
	deployment.Spec.Template = corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: containers,
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"app": app.Name},
		},
	}
	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{"app": app.Name},
	}
}

// DeleteDeployment deletes a Deployment if it exists
func DeleteDeployment(ctx context.Context, cli client.Client, app *basev1.App) error {
	var deployment v1.Deployment
	if err := cli.Get(ctx, client.ObjectKey{Name: DeploymentName(app), Namespace: app.Namespace}, &deployment); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if deployment.Name != "" {
		if err := cli.Delete(ctx, &deployment); err != nil {
			return err
		}
	}
	return nil
}
