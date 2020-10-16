package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"io/ioutil"
	"log"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig string
	filename   string
	dryrun     bool
)

func parse(kubeconfig, filename string, dryrun bool) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[YAML]:\n %q \n", string(data))

	kubecfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := kubernetes.NewForConfig(kubecfg)
	if err != nil {
		log.Fatal(err)
	}

	dynamicCfg, err := dynamic.NewForConfig(kubecfg)
	if err != nil {
		log.Fatal(err)
	}

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(data), 100)
	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			break
		}

		// gvk -> GroupVersionKind https://godoc.org/k8s.io/apimachinery/pkg/runtime/schema#GroupVersionKind
		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			log.Fatal(err)
		}

		unstructuredObj := &unstructured.Unstructured{
			Object: unstructuredMap,
		}

		groupRes, err := restmapper.GetAPIGroupResources(cfg.Discovery())
		if err != nil {
			log.Fatal(err)
		}

		mapper := restmapper.NewDiscoveryRESTMapper(groupRes)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			log.Fatal(err)
		}

		var dri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace("default")
			}
			dri = dynamicCfg.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = dynamicCfg.Resource(mapping.Resource)
		}

		if dryrun {
			log.Printf("[DryRun!] Successfully create k8s resource %s from yaml\n", unstructuredObj)
		} else {
			if _, err := dri.Create(context.Background(), unstructuredObj, metav1.CreateOptions{}); err != nil {
				log.Fatal(err)
			}
			log.Printf("Successfully create k8s resource %s from yaml\n", unstructuredObj)
		}
	}
	// readfile io EOF will break loop
	if err != io.EOF {
		log.Fatal("EOF error: ", err)
	}
}

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "kubeconfig of kubernetes cluster")
	flag.StringVar(&filename, "file", "", "kubernetes resource yaml")
	flag.BoolVar(&dryrun, "dryrun", false, "")
	flag.Parse()

	// yaml parser
	parse(kubeconfig, filename, dryrun)
}
