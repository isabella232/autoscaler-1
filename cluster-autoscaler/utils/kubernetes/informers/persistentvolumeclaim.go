package informers

import (
	"github.com/coreos/prometheus-operator/pkg/listwatch"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	time "time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	internalinterfaces "k8s.io/client-go/informers/internalinterfaces"
	kubernetes "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	cache "k8s.io/client-go/tools/cache"
)

type persistentVolumeClaimInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespaces       []string
}

func (f *persistentVolumeClaimInformer) defaultInformer(client kubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		listwatch.MultiNamespaceListerWatcher(nil, f.namespaces, []string{}, func(namespace string) cache.ListerWatcher {
			return &cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return client.CoreV1().PersistentVolumeClaims(namespace).List(options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return client.CoreV1().PersistentVolumeClaims(namespace).Watch(options)
				},
			}
		}),
		&corev1.Pod{},
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
}

func (f *persistentVolumeClaimInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&corev1.PersistentVolumeClaim{}, f.defaultInformer)
}

func (f *persistentVolumeClaimInformer) Lister() v1.PersistentVolumeClaimLister {
	return v1.NewPersistentVolumeClaimLister(f.Informer().GetIndexer())
}

func NewPersistentVolumeClaimInformer(f informers.SharedInformerFactory, ns []string) coreinformers.PersistentVolumeClaimInformer {
	var pvcInformer coreinformers.PersistentVolumeClaimInformer = &persistentVolumeClaimInformer{
		factory:          f,
		tweakListOptions: nil,
		namespaces:       ns,
	}
	return pvcInformer
}
