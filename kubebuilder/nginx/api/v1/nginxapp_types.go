/*
 * @Author: magician
 * @Date: 2025-03-18 17:03:14
 * @LastEditors: magician
 * @LastEditTime: 2025-03-18 17:04:34
 * @FilePath: /kubebuilder/nginx/api/v1/nginxapp_types.go
 * @Description:
 *
 * Copyright (c) 2025 by ${git_name_email}, All Rights Reserved.
 */
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NginxAppSpec struct {
	Replicas int32  `json:"replicas"`
	Image    string `json:"image"`
	Config   string `json:"config"`
}

type NginxAppStatus struct {
	LastBackup    string `json:"lastBackup,omitempty"`
	ServicePort   int32  `json:"servicePort,omitempty"`
	ReadyReplicas int32  `json:"readyReplicas"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type NginxApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NginxAppSpec   `json:"spec,omitempty"`
	Status NginxAppStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type NginxAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NginxApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NginxApp{}, &NginxAppList{})
}
