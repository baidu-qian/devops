/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	webv1 "example.com/m/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("NginxApp Controller", func() {
	const (
		NginxAppName      = "test-nginx"
		NginxAppNamespace = "default"
		timeout           = time.Second * 10
		interval          = time.Millisecond * 250
	)

	Context("When creating NginxApp", func() {
		It("Should create Deployment, Service, ConfigMap and Secret", func() {
			ctx := context.Background()

			nginxApp := &webv1.NginxApp{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "web.example.com/v1",
					Kind:       "NginxApp",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      NginxAppName,
					Namespace: NginxAppNamespace,
				},
				Spec: webv1.NginxAppSpec{
					Replicas: 1,
					Image:    "nginx:1.25",
					Config: `server {
						listen 8443 ssl;
						server_name nginx-test.example.com;
						ssl_certificate /etc/nginx/ssl/tls.crt;
						ssl_certificate_key /etc/nginx/ssl/tls.key;
						location / {
							root /usr/share/nginx/html;
							index index.html;
						}
					}`,
				},
			}

			// 创建 NginxApp
			Expect(k8sClient.Create(ctx, nginxApp)).Should(Succeed())

			// 检查 Deployment
			deploymentLookupKey := types.NamespacedName{Name: NginxAppName, Namespace: NginxAppNamespace}
			createdDeployment := &appsv1.Deployment{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, deploymentLookupKey, createdDeployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdDeployment.Spec.Template.Spec.Containers[0].Image).Should(Equal("nginx:1.25"))
			Expect(*createdDeployment.Spec.Replicas).Should(Equal(int32(1)))

			// 检查 Service
			serviceLookupKey := types.NamespacedName{Name: NginxAppName, Namespace: NginxAppNamespace}
			createdService := &corev1.Service{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, serviceLookupKey, createdService)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdService.Spec.Ports[0].Port).Should(Equal(int32(443)))

			// 检查 ConfigMap
			configMapLookupKey := types.NamespacedName{Name: NginxAppName + "-config", Namespace: NginxAppNamespace}
			createdConfigMap := &corev1.ConfigMap{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, configMapLookupKey, createdConfigMap)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// 检查 Secret
			secretLookupKey := types.NamespacedName{Name: NginxAppName + "-tls", Namespace: NginxAppNamespace}
			createdSecret := &corev1.Secret{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, createdSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When updating NginxApp", func() {
		It("Should update Deployment image", func() {
			ctx := context.Background()

			// 获取现有的 NginxApp
			nginxApp := &webv1.NginxApp{}
			nginxAppKey := types.NamespacedName{Name: NginxAppName, Namespace: NginxAppNamespace}
			Expect(k8sClient.Get(ctx, nginxAppKey, nginxApp)).Should(Succeed())

			// 更新镜像
			nginxApp.Spec.Image = "nginx:1.26"
			Expect(k8sClient.Update(ctx, nginxApp)).Should(Succeed())

			// 验证 Deployment 是否更新
			deploymentLookupKey := types.NamespacedName{Name: NginxAppName, Namespace: NginxAppNamespace}
			updatedDeployment := &appsv1.Deployment{}

			Eventually(func() string {
				err := k8sClient.Get(ctx, deploymentLookupKey, updatedDeployment)
				if err != nil {
					return ""
				}
				return updatedDeployment.Spec.Template.Spec.Containers[0].Image
			}, timeout, interval).Should(Equal("nginx:1.26"))
		})
	})

	Context("When deleting NginxApp", func() {
		It("Should delete all related resources", func() {
			ctx := context.Background()

			// 删除 NginxApp
			nginxApp := &webv1.NginxApp{}
			nginxAppKey := types.NamespacedName{Name: NginxAppName, Namespace: NginxAppNamespace}
			Expect(k8sClient.Get(ctx, nginxAppKey, nginxApp)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, nginxApp)).Should(Succeed())

			// 验证资源是否被删除
			Eventually(func() bool {
				err := k8sClient.Get(ctx, nginxAppKey, &webv1.NginxApp{})
				return err != nil
			}, timeout, interval).Should(BeTrue())

			// 验证 Deployment 是否被删除
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: NginxAppName, Namespace: NginxAppNamespace}, &appsv1.Deployment{})
				return err != nil
			}, timeout, interval).Should(BeTrue())

			// 验证 Service 是否被删除
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: NginxAppName, Namespace: NginxAppNamespace}, &corev1.Service{})
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})
