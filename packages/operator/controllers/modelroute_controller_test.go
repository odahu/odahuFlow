//
//    Copyright 2019 EPAM Systems
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

package controllers_test

import (
	v1alpha3_istio_api "github.com/aspenmesh/istio-client-go/pkg/apis/networking/v1alpha3"
	odahuflowv1alpha1 "github.com/odahu/odahu-flow/packages/operator/api/v1alpha1"
	. "github.com/odahu/odahu-flow/packages/operator/controllers"
	"github.com/odahu/odahu-flow/packages/operator/pkg/config"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sync"
	"testing"
	"time"
)

const (
	mrName          = "test-mr"
	mrURL           = "/test/url"
	timeout         = time.Second * 5
	istioIngressSvc = "istio-ingressgateway.istio-system.svc.cluster.local"
)

var (
	testNamespace = "default"
	md1           = &odahuflowv1alpha1.ModelDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "md-1-for-tests",
			Namespace: testNamespace,
		},
		Spec: odahuflowv1alpha1.ModelDeploymentSpec{
			Image: "test/image:1",
		},
		Status: odahuflowv1alpha1.ModelDeploymentStatus{
			HostHeader: "md-1-for-tests.default.svc",
			State:      odahuflowv1alpha1.ModelDeploymentStateReady,
		},
	}
	md2 = &odahuflowv1alpha1.ModelDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "md-2-for-tests",
			Namespace: testNamespace,
		},
		Spec: odahuflowv1alpha1.ModelDeploymentSpec{
			Image: "test/image:1",
		},
		Status: odahuflowv1alpha1.ModelDeploymentStatus{
			HostHeader: "md-2-for-tests.default.svc",
			State:      odahuflowv1alpha1.ModelDeploymentStateReady,
		},
	}
	mdNotReady = &odahuflowv1alpha1.ModelDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "md-not-ready",
			Namespace: testNamespace,
		},
		Spec: odahuflowv1alpha1.ModelDeploymentSpec{
			Image: "test/image:1",
		},
		Status: odahuflowv1alpha1.ModelDeploymentStatus{
			HostHeader: "",
		},
	}
	c                    client.Client
	routeExpectedRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: mrName, Namespace: testNamespace}}
	mrKey                = types.NamespacedName{Name: mrName, Namespace: testNamespace}
)

func setUp(g *GomegaWithT) (chan struct{}, *sync.WaitGroup, chan reconcile.Request) {
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())
	c = mgr.GetClient()

	md1.ObjectMeta.ResourceVersion = ""
	md2.ObjectMeta.ResourceVersion = ""
	mdNotReady.ObjectMeta.ResourceVersion = ""

	if err := c.Create(context.TODO(), md1); err != nil {
		panic(err)
	}

	if err := c.Create(context.TODO(), md2); err != nil {
		panic(err)
	}

	if err := c.Create(context.TODO(), mdNotReady); err != nil {
		panic(err)
	}

	requests := make(chan reconcile.Request)
	rw := NewReconcilerWrapper(NewModelRouteReconciler(mgr, *config.NewDefaultConfig()), requests)

	g.Expect(rw.SetupWithManager(mgr)).NotTo(HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	return stopMgr, mgrStopped, requests
}

func teardown(stopMgr chan struct{}, mgrStopped *sync.WaitGroup) {
	_ = c.Delete(context.TODO(), mdNotReady)
	_ = c.Delete(context.TODO(), md2)
	_ = c.Delete(context.TODO(), md1)

	close(stopMgr)
	mgrStopped.Wait()
}

func TestBasicReconcile(t *testing.T) {
	g := NewGomegaWithT(t)
	stopMgr, mgrStopped, requests := setUp(g)
	defer teardown(stopMgr, mgrStopped)

	weight := int32(100)
	mr := &odahuflowv1alpha1.ModelRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mrName,
			Namespace: testNamespace,
		},
		Spec: odahuflowv1alpha1.ModelRouteSpec{
			URLPrefix: mrURL,
			Mirror:    &md1.ObjectMeta.Name,
			ModelDeploymentTargets: []odahuflowv1alpha1.ModelDeploymentTarget{
				{
					Name:   md2.ObjectMeta.Name,
					Weight: &weight,
				},
			},
		},
	}

	err := c.Create(context.TODO(), mr)
	g.Expect(err).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), mr)

	g.Eventually(requests, timeout).Should(Receive(Equal(routeExpectedRequest)))
	g.Eventually(requests, timeout).Should(Receive(Equal(routeExpectedRequest)))

	g.Expect(c.Get(context.TODO(), mrKey, mr)).ToNot(HaveOccurred())
	g.Eventually(mr.Status.State, timeout).Should(Equal(odahuflowv1alpha1.ModelRouteStateReady))

	vs := &v1alpha3_istio_api.VirtualService{}
	vsKey := types.NamespacedName{Name: VirtualServiceName(mr), Namespace: testNamespace}
	g.Expect(c.Get(context.TODO(), vsKey, vs)).ToNot(HaveOccurred())

	g.Expect(vs.Spec.Http).To(HaveLen(1))

	for _, host := range vs.Spec.Http {
		g.Expect(host.Mirror).ToNot(BeNil())
		g.Expect(host.Mirror.Host).To(Equal(md1.Name))

		g.Expect(host.Route).To(HaveLen(1))
		g.Expect(host.Route[0].Destination.Host).To(Equal(istioIngressSvc))
		g.Expect(host.Route[0].Weight).To(Equal(weight))
		g.Expect(host.Route[0].Headers.Request.Set["Host"]).To(Equal(md2.Status.HostHeader))
	}
}

func TestEmptyMirror(t *testing.T) {
	g := NewGomegaWithT(t)
	stopMgr, mgrStopped, requests := setUp(g)
	defer teardown(stopMgr, mgrStopped)

	weight := int32(100)
	mr := &odahuflowv1alpha1.ModelRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mrName,
			Namespace: testNamespace,
		},
		Spec: odahuflowv1alpha1.ModelRouteSpec{
			URLPrefix: mrURL,
			ModelDeploymentTargets: []odahuflowv1alpha1.ModelDeploymentTarget{
				{
					Name:   md2.ObjectMeta.Name,
					Weight: &weight,
				},
			},
		},
	}

	err := c.Create(context.TODO(), mr)
	g.Expect(err).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), mr)

	g.Eventually(requests, timeout).Should(Receive(Equal(routeExpectedRequest)))
	g.Eventually(requests, timeout).Should(Receive(Equal(routeExpectedRequest)))

	g.Expect(c.Get(context.TODO(), mrKey, mr)).ToNot(HaveOccurred())
	g.Expect(mr.Status.State).To(Equal(odahuflowv1alpha1.ModelRouteStateReady))

	vs := &v1alpha3_istio_api.VirtualService{}
	vsKey := types.NamespacedName{Name: VirtualServiceName(mr), Namespace: testNamespace}
	g.Expect(c.Get(context.TODO(), vsKey, vs)).ToNot(HaveOccurred())

	g.Expect(vs.Spec.Http).To(HaveLen(1))

	for _, host := range vs.Spec.Http {
		g.Expect(host.Mirror).To(BeNil())
	}
}

func TestNotReadyEmptyMirror(t *testing.T) {
	g := NewGomegaWithT(t)
	stopMgr, mgrStopped, requests := setUp(g)
	defer teardown(stopMgr, mgrStopped)

	weight := int32(100)
	mr := &odahuflowv1alpha1.ModelRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mrName,
			Namespace: testNamespace,
		},
		Spec: odahuflowv1alpha1.ModelRouteSpec{
			URLPrefix: mrURL,
			Mirror:    &mdNotReady.ObjectMeta.Name,
			ModelDeploymentTargets: []odahuflowv1alpha1.ModelDeploymentTarget{
				{
					Name:   md2.ObjectMeta.Name,
					Weight: &weight,
				},
			},
		},
	}

	err := c.Create(context.TODO(), mr)
	g.Expect(err).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), mr)

	g.Eventually(requests, timeout).Should(Receive(Equal(routeExpectedRequest)))
	g.Eventually(requests, timeout).Should(Receive(Equal(routeExpectedRequest)))

	g.Expect(c.Get(context.TODO(), mrKey, mr)).ToNot(HaveOccurred())
	g.Expect(mr.Status.State).To(Equal(odahuflowv1alpha1.ModelRouteStateProcessing))

	vs := &v1alpha3_istio_api.VirtualService{}
	vsKey := types.NamespacedName{Name: VirtualServiceName(mr), Namespace: testNamespace}
	g.Expect(c.Get(context.TODO(), vsKey, vs)).ToNot(HaveOccurred())

	g.Expect(vs.Spec.Http).To(HaveLen(1))

	for _, host := range vs.Spec.Http {
		g.Expect(host.Mirror).To(BeNil())
	}
}

func TestMultipleTargets(t *testing.T) {
	g := NewGomegaWithT(t)
	stopMgr, mgrStopped, requests := setUp(g)
	defer teardown(stopMgr, mgrStopped)

	weight := int32(50)
	mr := &odahuflowv1alpha1.ModelRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mrName,
			Namespace: testNamespace,
		},
		Spec: odahuflowv1alpha1.ModelRouteSpec{
			URLPrefix: mrURL,
			ModelDeploymentTargets: []odahuflowv1alpha1.ModelDeploymentTarget{
				{
					Name:   md1.ObjectMeta.Name,
					Weight: &weight,
				},
				{
					Name:   md2.ObjectMeta.Name,
					Weight: &weight,
				},
			},
		},
	}

	err := c.Create(context.TODO(), mr)
	g.Expect(err).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), mr)

	g.Eventually(requests, timeout).Should(Receive(Equal(routeExpectedRequest)))
	g.Eventually(requests, timeout).Should(Receive(Equal(routeExpectedRequest)))

	g.Expect(c.Get(context.TODO(), mrKey, mr)).ToNot(HaveOccurred())
	g.Expect(mr.Status.State).To(Equal(odahuflowv1alpha1.ModelRouteStateReady))

	vs := &v1alpha3_istio_api.VirtualService{}
	vsKey := types.NamespacedName{Name: VirtualServiceName(mr), Namespace: testNamespace}
	g.Expect(c.Get(context.TODO(), vsKey, vs)).ToNot(HaveOccurred())

	g.Expect(vs.Spec.Http).To(HaveLen(1))

	for _, host := range vs.Spec.Http {
		g.Expect(host.Route).To(HaveLen(2))

		g.Expect(host.Route[0].Destination.Host).To(Equal(istioIngressSvc))
		g.Expect(host.Route[0].Weight).To(Equal(weight))
		g.Expect(host.Route[0].Headers.Request.Set["Host"]).To(Equal(md1.Status.HostHeader))

		g.Expect(host.Route[1].Destination.Host).To(Equal(istioIngressSvc))
		g.Expect(host.Route[1].Weight).To(Equal(weight))
		g.Expect(host.Route[1].Headers.Request.Set["Host"]).To(Equal(md2.Status.HostHeader))
	}
}
