package informers

import (
	"github.com/coreos/prometheus-operator/pkg/listwatch"
	"k8s.io/client-go/informers"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	time "time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	internalinterfaces "k8s.io/client-go/informers/internalinterfaces"
	kubernetes "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/apps/v1"
	cache "k8s.io/client-go/tools/cache"
)

// ReplicaSetInformer provides access to a shared informer and lister for
// ReplicaSets.
type ReplicaSetInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.ReplicaSetLister
}

type replicaSetInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespaces       []string
}

func (f *replicaSetInformer) defaultInformer(client kubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		listwatch.MultiNamespaceListerWatcher(nil, f.namespaces, []string{}, func(namespace string) cache.ListerWatcher {
			return &cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return client.AppsV1().ReplicaSets(namespace).List(options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return client.AppsV1().ReplicaSets(namespace).Watch(options)
				},
			}
		}),
		&appsv1.ReplicaSet{},
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
}

func (f *replicaSetInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&appsv1.ReplicaSet{}, f.defaultInformer)
}

func (f *replicaSetInformer) Lister() v1.ReplicaSetLister {
	return v1.NewReplicaSetLister(f.Informer().GetIndexer())
}

func NewReplicaSetInformer(f informers.SharedInformerFactory, ns []string) appsinformers.ReplicaSetInformer {
	var replicaSetInformer appsinformers.ReplicaSetInformer = &replicaSetInformer{
		factory:          f,
		tweakListOptions: nil,
		namespaces:       ns,
	}
	return replicaSetInformer
}
