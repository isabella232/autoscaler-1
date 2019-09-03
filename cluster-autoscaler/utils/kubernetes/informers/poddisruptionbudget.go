package informers

import (
	"github.com/coreos/prometheus-operator/pkg/listwatch"
	"k8s.io/client-go/informers"
	policyinformers "k8s.io/client-go/informers/policy/v1beta1"
	"k8s.io/client-go/listers/policy/v1beta1"
	"time"

	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// PodDisruptionBudgetInformer provides access to a shared informer and lister for
// PodDisruptionBudgets.
type PodDisruptionBudgetInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1beta1.PodDisruptionBudgetLister
}

type podDisruptionBudgetInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespaces       []string
}

func (f *podDisruptionBudgetInformer) defaultInformer(client kubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		listwatch.MultiNamespaceListerWatcher(nil, f.namespaces, []string{}, func(namespace string) cache.ListerWatcher {
			return &cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return client.PolicyV1beta1().PodDisruptionBudgets(namespace).List(options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return client.PolicyV1beta1().PodDisruptionBudgets(namespace).Watch(options)
				},
			}
		}),
		&policyv1beta1.PodDisruptionBudget{},
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
}

func (f *podDisruptionBudgetInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&policyv1beta1.PodDisruptionBudget{}, f.defaultInformer)
}

func (f *podDisruptionBudgetInformer) Lister() v1beta1.PodDisruptionBudgetLister {
	return v1beta1.NewPodDisruptionBudgetLister(f.Informer().GetIndexer())
}

func NewPodDisruptionBudgetInformer(f informers.SharedInformerFactory, ns []string) policyinformers.PodDisruptionBudgetInformer {
	var podDisruptionBudgetInformer policyinformers.PodDisruptionBudgetInformer = &podDisruptionBudgetInformer{
		factory:          f,
		tweakListOptions: nil,
		namespaces:       ns,
	}
	return podDisruptionBudgetInformer
}
