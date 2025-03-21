/*
 * @Descripttion:
 * @Author: Magician
 * @version:
 * @Date: 2025-03-19 23:34:22
 * @LastEditors: Magician
 * @LastEditTime: 2025-03-21 16:13:07
 */
package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	webv1 "example.com/m/api/v1"
	appsv1 "k8s.io/api/apps/v1"
)

type NginxAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *NginxAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	nginxApp := &webv1.NginxApp{}
	if err := r.Get(ctx, req.NamespacedName, nginxApp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling NginxApp", "namespace", req.Namespace, "name", req.Name)

	// 处理ConfigMap
	if err := r.handleConfigMap(ctx, nginxApp); err != nil {
		logger.Error(err, "Failed to handle ConfigMap")
		return ctrl.Result{}, err
	}

	// 处理Secret
	if err := r.handleSecret(ctx, nginxApp); err != nil {
		logger.Error(err, "Failed to handle Secret")
		return ctrl.Result{}, err
	}

	// 处理Deployment
	if err := r.handleDeployment(ctx, nginxApp); err != nil {
		logger.Error(err, "Failed to handle Deployment")
		return ctrl.Result{}, err
	}

	// 处理Service
	if err := r.handleService(ctx, nginxApp); err != nil {
		logger.Error(err, "Failed to handle Service")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NginxAppReconciler) handleConfigMap(ctx context.Context, nginxApp *webv1.NginxApp) error {
	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: nginxApp.Namespace,
		Name:      nginxApp.Name + "-config",
	}, configMap)

	// 在备份操作前添加日志
	logger := log.FromContext(ctx)
	if err == nil && configMap.Data["web.conf"] != nginxApp.Spec.Config {
		logger.Info("Config changed, creating backup",
			"old_config", configMap.Data["web.conf"],
			"new_config", nginxApp.Spec.Config)
		backupName := fmt.Sprintf("%s-backup-%d", nginxApp.Name, time.Now().Unix())
		backupCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backupName,
				Namespace: nginxApp.Namespace,
				Labels:    map[string]string{"app": nginxApp.Name},
			},
			Data: map[string]string{
				"web.conf": configMap.Data["web.conf"],
			},
		}
		if err := r.Create(ctx, backupCM); err != nil {
			return err
		}
	}

	// 创建/更新主ConfigMap
	desiredCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nginxApp.Name + "-config",
			Namespace: nginxApp.Namespace,
		},
		Data: map[string]string{
			"web.conf": nginxApp.Spec.Config,
		},
	}

	if err := ctrl.SetControllerReference(nginxApp, desiredCM, r.Scheme); err != nil {
		return err
	}

	if errors.IsNotFound(err) {
		return r.Create(ctx, desiredCM)
	} else if err == nil {
		return r.Update(ctx, desiredCM)
	}
	return err
}

func (r *NginxAppReconciler) handleSecret(ctx context.Context, nginxApp *webv1.NginxApp) error {
	secret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: nginxApp.Namespace,
		Name:      nginxApp.Name + "-tls",
	}, secret)

	// 如果Secret不存在，创建一个空的Secret（实际使用时需要填入真实的证书数据）
	desiredSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nginxApp.Name + "-tls",
			Namespace: nginxApp.Namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": []byte{}, // 需要替换为实际的证书内容
			"tls.key": []byte{}, // 需要替换为实际的私钥内容
		},
	}

	if err := ctrl.SetControllerReference(nginxApp, desiredSecret, r.Scheme); err != nil {
		return err
	}

	if errors.IsNotFound(err) {
		return r.Create(ctx, desiredSecret)
	} else if err == nil {
		return r.Update(ctx, desiredSecret)
	}
	return err
}

func (r *NginxAppReconciler) handleDeployment(ctx context.Context, nginxApp *webv1.NginxApp) error {
	logger := log.FromContext(ctx)
	deploy := &appsv1.Deployment{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: nginxApp.Namespace,
		Name:      nginxApp.Name,
	}, deploy)

	logger.Info("Handling deployment", "image", nginxApp.Spec.Image)

	desiredDeploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nginxApp.Name,
			Namespace: nginxApp.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &nginxApp.Spec.Replicas,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": nginxApp.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": nginxApp.Name,
					},
					Annotations: map[string]string{
						"image-version": nginxApp.Spec.Image,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "nginx",
						Image: nginxApp.Spec.Image,
						Ports: []corev1.ContainerPort{
							{ContainerPort: 8443},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "config",
								MountPath: "/etc/nginx/conf.d/web.conf",
								SubPath:   "web.conf",
							},
							{
								Name:      "tls",
								MountPath: "/etc/nginx/ssl",
								ReadOnly:  true,
							},
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								TCPSocket: &corev1.TCPSocketAction{
									Port: intstr.FromInt(8443),
								},
							},
							InitialDelaySeconds: 15,
							PeriodSeconds:       20,
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: nginxApp.Name + "-config",
									},
								},
							},
						},
						{
							Name: "tls",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: nginxApp.Name + "-tls",
								},
							},
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(nginxApp, desiredDeploy, r.Scheme); err != nil {
		return err
	}

	if errors.IsNotFound(err) {
		logger.Info("Creating new deployment")
		return r.Create(ctx, desiredDeploy)
	} else if err == nil {
		logger.Info("Patching existing deployment", "current_image", deploy.Spec.Template.Spec.Containers[0].Image, "desired_image", nginxApp.Spec.Image)
		patch := client.MergeFrom(deploy.DeepCopy())
		deploy.Spec = desiredDeploy.Spec
		return r.Patch(ctx, deploy, patch)
	}
	return err
}

func (r *NginxAppReconciler) handleService(ctx context.Context, nginxApp *webv1.NginxApp) error {
	svc := &corev1.Service{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: nginxApp.Namespace,
		Name:      nginxApp.Name,
	}, svc)

	desiredSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nginxApp.Name,
			Namespace: nginxApp.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": nginxApp.Name},
			Ports: []corev1.ServicePort{{
				Port:       443,
				TargetPort: intstr.FromInt(8443),
			}},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	if err := ctrl.SetControllerReference(nginxApp, desiredSvc, r.Scheme); err != nil {
		return err
	}

	if errors.IsNotFound(err) {
		return r.Create(ctx, desiredSvc)
	} else if err == nil {
		return r.Update(ctx, desiredSvc)
	}
	return err
}

func (r *NginxAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webv1.NginxApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
