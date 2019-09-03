package informers

import (
	"github.com/coreos/prometheus-operator/pkg/listwatch"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// ServiceInformer provides access to a shared informer and lister for
// Services.
type ServiceInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.ServiceLister
}

type serviceInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespaces       []string
}

func (f *serviceInformer) defaultInformer(client kubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		listwatch.MultiNamespaceListerWatcher(nil, f.namespaces, []string{}, func(namespace string) cache.ListerWatcher {
			return &cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return client.CoreV1().Services(namespace).List(options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return client.CoreV1().Services(namespace).Watch(options)
				},
			}
		}),
		&corev1.Pod{},
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
}

func (f *serviceInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&corev1.Service{}, f.defaultInformer)
}

func (f *serviceInformer) Lister() v1.ServiceLister {
	return v1.NewServiceLister(f.Informer().GetIndexer())
}

func NewServiceInformer(f informers.SharedInformerFactory, ns []string) coreinformers.ServiceInformer {
	var serviceInformer coreinformers.ServiceInformer = &serviceInformer{
		factory:          f,
		tweakListOptions: nil,
		namespaces:       ns,
	}
	return serviceInformer
}
