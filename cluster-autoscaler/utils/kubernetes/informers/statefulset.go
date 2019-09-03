package informers

import (
	"github.com/coreos/prometheus-operator/pkg/listwatch"
	"k8s.io/client-go/informers"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
)

// StatefulSetInformer provides access to a shared informer and lister for
// StatefulSets.
type StatefulSetInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.StatefulSetLister
}

type statefulSetInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespaces       []string
}

func (f *statefulSetInformer) defaultInformer(client kubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		listwatch.MultiNamespaceListerWatcher(nil, f.namespaces, []string{}, func(namespace string) cache.ListerWatcher {
			return &cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return client.AppsV1().StatefulSets(namespace).List(options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return client.AppsV1().StatefulSets(namespace).Watch(options)
				},
			}
		}),
		&appsv1.StatefulSet{},
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
}

func (f *statefulSetInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&appsv1.StatefulSet{}, f.defaultInformer)
}

func (f *statefulSetInformer) Lister() v1.StatefulSetLister {
	return v1.NewStatefulSetLister(f.Informer().GetIndexer())
}

func NewStatefulSetInformer(f informers.SharedInformerFactory, ns []string) appsinformers.StatefulSetInformer {
	var statefulSetInformer appsinformers.StatefulSetInformer = &statefulSetInformer{
		factory:          f,
		tweakListOptions: nil,
		namespaces:       ns,
	}
	return statefulSetInformer
}
