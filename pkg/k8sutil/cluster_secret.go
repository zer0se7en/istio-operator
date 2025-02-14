/*
Copyright 2021 Cisco Systems, Inc. and/or its affiliates.

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

package k8sutil

import (
	"context"
	"net"
	"net/url"
	"strings"

	"emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	k8sclientapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	ctrl "sigs.k8s.io/controller-runtime"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

	clusterregistryv1alpha1 "github.com/cisco-open/cluster-registry-controller/api/v1alpha1"
)

func GetExternalAddressOfAPIServer(kubeConfig *rest.Config) (string, error) {
	d, err := discovery.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		return "", errors.WithStackIf(err)
	}

	v := &metav1.APIVersions{}
	err = d.RESTClient().Get().AbsPath(d.LegacyPrefix).Do(context.TODO()).Into(v)
	if err != nil {
		return "", errors.WithStackIf(err)
	}

	for _, addr := range v.ServerAddressByClientCIDRs {
		if addr.ClientCIDR == (&net.IPNet{
			IP:   net.IPv4zero,
			Mask: net.IPv4Mask(0, 0, 0, 0),
		}).String() {
			return (&url.URL{
				Scheme: "https",
				Host:   addr.ServerAddress,
			}).String(), nil
		}
	}

	return "", errors.New("could not determine external apiserver address")
}

func GetReaderSecretForCluster(ctx context.Context, kubeClient client.Client, kubeConfig *rest.Config, clusterName string, secretRef types.NamespacedName, saRef types.NamespacedName, apiServerEndpointAddress string, clusterRegistryAPIEnabled bool) (*corev1.Secret, error) {
	sa := &corev1.ServiceAccount{}
	err := kubeClient.Get(ctx, saRef, sa)
	if err != nil {
		return nil, errors.WithStackIf(err)
	}

	clientSet, _ := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
	// After K8s v1.24, Secret objects containing ServiceAccount tokens are no longer auto-generated, so we will have to manually create Secret in order to get the token.
	// Reference: https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-1.24.md#no-really-you-must-read-this-before-you-upgrade
	var secretObj *corev1.Secret

	readerSecretName := saRef.Name + "-token"
	if len(sa.Secrets) != 0 {
		readerSecretName = sa.Secrets[0].Name
	}

	secretObj, err = clientSet.CoreV1().Secrets(saRef.Namespace).Get(ctx, readerSecretName, metav1.GetOptions{})
	if err != nil &&
		k8sErrors.IsNotFound(err) {
		readerSATokenSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      readerSecretName,
				Namespace: saRef.Namespace,
				Annotations: map[string]string{
					"kubernetes.io/service-account.name": saRef.Name,
				},
			},
			Type: "kubernetes.io/service-account-token",
		}

		secretObj, err = clientSet.CoreV1().Secrets(saRef.Namespace).Create(ctx, &readerSATokenSecret, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.WrapIfWithDetails(err, "creating kubernetes secret failed", "namespace", saRef.Namespace, "secret", readerSecretName)
		}
	} else if err != nil {
		return nil, errors.WrapIfWithDetails(
			err,
			"retrieving kubernetes secret failed with unexpected error",
			"namespace", saRef.Namespace,
			"secret", readerSecretName,
		)
	}

	// fetch CA certificate and token from secret associated with reader SA
	saToken := string(secretObj.Data["token"])
	caData := secretObj.Data["ca.crt"]

	if clusterRegistryAPIEnabled {
		cluster, err := GetLocalCluster(ctx, kubeClient)
		if err != nil {
			return nil, errors.WithStackIf(err)
		}

		// add overrides specified in the cluster resource without network specified
		endpoint := GetEndpointForClusterByNetwork(cluster, "")
		if endpoint.ServerAddress != "" {
			apiServerEndpointAddress = endpoint.ServerAddress
		}
		if len(endpoint.CABundle) > 0 {
			caData = append(append(caData, []byte("\n")...), endpoint.CABundle...)
		}
	}

	// try to get external ip address from api server
	if apiServerEndpointAddress == "" {
		apiServerEndpointAddress, _ = GetExternalAddressOfAPIServer(kubeConfig)
	}

	// try to get endpoint from used kubeconfig
	if apiServerEndpointAddress == "" {
		apiServerEndpointAddress = kubeConfig.Host
	}

	kubeconfig, err := GetKubeconfigWithSAToken(secretRef.Name, sa.GetName(), apiServerEndpointAddress, caData, saToken)
	if err != nil {
		return nil, errors.WithStackIf(err)
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretRef.Name,
			Namespace: secretRef.Namespace,
		},
		Type: clusterregistryv1alpha1.SecretTypeClusterRegistry,
		Data: map[string][]byte{
			clusterName: []byte(kubeconfig),
		},
	}, nil
}

func GetKubeconfigWithSAToken(name, username, endpointURL string, caData []byte, saToken string) (string, error) {
	if !strings.Contains(endpointURL, "//") {
		endpointURL = "//" + endpointURL
	}
	u, err := url.Parse(endpointURL)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}

	config := k8sclientapiv1.Config{
		APIVersion: k8sclientapiv1.SchemeGroupVersion.Version,
		Kind:       "Config",
		Clusters: []k8sclientapiv1.NamedCluster{
			{
				Name: name,
				Cluster: k8sclientapiv1.Cluster{
					CertificateAuthorityData: caData,
					Server:                   u.String(),
				},
			},
		},
		Contexts: []k8sclientapiv1.NamedContext{
			{
				Name: name,
				Context: k8sclientapiv1.Context{
					Cluster:  name,
					AuthInfo: username,
				},
			},
		},
		CurrentContext: name,
		AuthInfos: []k8sclientapiv1.NamedAuthInfo{
			{
				Name: username,
				AuthInfo: k8sclientapiv1.AuthInfo{
					Token: saToken,
				},
			},
		},
	}

	y, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}

	return string(y), nil
}

func GetEndpointForClusterByNetwork(cluster *clusterregistryv1alpha1.Cluster, networkName string) clusterregistryv1alpha1.KubernetesAPIEndpoint {
	var endpoint clusterregistryv1alpha1.KubernetesAPIEndpoint

	for _, apiEndpoint := range cluster.Spec.KubernetesAPIEndpoints {
		if apiEndpoint.ClientNetwork == networkName {
			endpoint = apiEndpoint

			break
		}
		// use for every network if the endpoint is not network specific
		if apiEndpoint.ClientNetwork == "" {
			endpoint = apiEndpoint
		}
	}

	return endpoint
}
