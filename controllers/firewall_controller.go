/*


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

package controllers

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batchv1 "operator-sample2/api/v1"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

// FireWallReconciler reconciles a FireWall object
type FireWallReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=firewalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=firewalls/status,verbs=get;update;patch

func (r *FireWallReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("firewall", req.NamespacedName)

	//making request to prometheus using golang client
	clientP, err := api.NewClient(api.Config{
		Address: "http://localhost:9090",
	})
	if err == nil {
		v1api := v1.NewAPI(clientP)
		ctxWT, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		result, warnings, err := v1api.Query(ctxWT, "kube_pod_info{pod=~\"nginx.*\"}", time.Now())
		if err == nil {
			if len(warnings) > 0 {
				fmt.Printf("Warnings: %v\n", warnings)
			}
			fmt.Printf("Prometheus Result:\n%v\n", result)
		} else {
			fmt.Printf("Error querying Prometheus: %v\n", err)
		}

	} else {
		fmt.Println("Error creating client: ", err)
	}

	// Prendo l'istanza di fireWall che ha Name e Namespace presenti nella richiesta
	var fireWall batchv1.FireWall
	if err := r.Get(ctx, req.NamespacedName, &fireWall); err != nil {
		log.Error(err, "unable to fetch fireWall")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Prendo l'istanza di Deployment che ha il Name definito dalla variabile del fireWall
	deploy := appsv1.Deployment{}
	DeployNamespacedName := types.NamespacedName{
		Namespace: "default",
		Name:      fireWall.Spec.DeploymentName,
	}
	// Prendo l'istanza di deployment che ha Name e Namespace presenti nella richiesta
	err = r.Get(ctx, DeployNamespacedName, &deploy)

	if err == nil {
		fmt.Println("Il numero di repliche del deploy ", deploy.Name, " è: ", deploy.Status.Replicas)
		if fireWall.Status.DeploymentReplicas == deploy.Status.Replicas {
			return ctrl.Result{}, nil
		}
		// Se arriva qui significa che è cambiato il numero di repliche

		// aggiorno la variabile di loadBalancer dopo aver preso l'istanza
		var loadBalancer batchv1.LoadBalancer
		NamespacedName := types.NamespacedName{
			Namespace: "default",
			Name:      fireWall.Spec.SecondOperatorName,
		}
		if err := r.Get(ctx, NamespacedName, &loadBalancer); err != nil {
			log.Info("unable to fetch loadBalancer in fireWall_controller")
			if client.IgnoreNotFound(err) == nil {
				// se non ha trovato il loadBalancer aggiorna comunque lo status del fireWall
				fireWall.Status.DeploymentReplicas = deploy.Status.Replicas
				if err := r.Status().Update(ctx, &fireWall); err != nil {
					log.Error(err, "unable to update fireWall status")
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		loadBalancer.Status.DeploymentReplicas = deploy.Status.Replicas
		//adesso aggiorno il loadBalancer prima così se c'è un errore si ferma prima di aggiornare
		//lo status del fireWall e alla prossima chiamata lo stato sarà ancora inconsistente e riproverà a modificare loadBalancer
		if err := r.Update(ctx, &loadBalancer, &client.UpdateOptions{}); err != nil {
			log.Error(err, "unable to update loadBalancer status in fireWall_controller")
			return ctrl.Result{}, err
		}

		//aggiorno lo stato di fireWall
		fireWall.Status.DeploymentReplicas = deploy.Status.Replicas
		if err := r.Update(ctx, &fireWall); err != nil {
			log.Error(err, "unable to update fireWall status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	} else {
		log.Info("unable to fetch deployment")
		return ctrl.Result{}, nil
	}
}

func (r *FireWallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	/*
		Invece di fare il Complete() sul controller, come viene fatto di default, chiamo il Build
		che mi permette di ottenere l'oggetto controller sul quale posso chiamare il watch().
		N.B. il Complete sul controller altro non fa che chiamare il Build (il quale torna il controller
		e un errore se c'è stato) e tornare un eventuale errore scartando l'oggetto controller.
	*/
	c, err := ctrl.NewControllerManagedBy(mgr).For(&batchv1.FireWall{}).Build(r)

	if err == nil && c != nil {
		/*
			la funzione Watch() prende in ingresso tre parametri:
			1) 	La sorgente degli eventi (in questo caso stiamo osservando i Deployment,
				quindi tutti gli eventi su Deployment faranno scattare l'handler).
			2)	L'EventHandler che accoda le reconcile.Request in risposta all'evento, in altre parole
				chiama la funzione reconcile() del controller su cui stiamo chiamando la Watch().
				N.B. EnqueueRequestForObject{}  accoda le richieste (Requests) con il Name e il Namespace (per abbreviare N&N)
					della sorgente degli eventi (nel nostro caso Deployment). Se si vuole accodare le richieste
					con un diverso N&N si possono usare EnqueueRequestForOwner (N&N del diretto possessore della sorgente)
					oppure EnqueueRequestsFromMapFunc (N&N di un oggetto differente definito da noi). Per maggiori info e
					esempi vedere la documentazione e gli esempi qui (https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/handler/example_test.go)
			3)	Un oggetto predicate che permette di filtrare gli eventi prima che vengano passati all'EventHandler.
				In questo esempio filtro solo gli eventi dovuti ad Update di Deployment, ma il filtro può essere applicato
				anche alla creazione (CreateFunc), eliminazione (DeleteFunc) e ad una funzione generica (GenericFunc)
				N.B	essendo una modifica l'evento e mi permette di accedere sia all'istanza vecchia con e.MetaOld che alla
					nuova con e.MetaNew
		*/
		/*
			errWatch := c.Watch(
				&source.Kind{Type: &appsv1.Deployment{}},
				&handler.EnqueueRequestForObject{},
				predicate.Funcs{
					UpdateFunc: func(e event.UpdateEvent) bool {
						labelsMap := e.MetaNew.GetLabels()
						fmt.Println("DeploymentLabelKey: ", utils.DeploymentLabelKey)
						fmt.Println("DeploymentLabelValue: ", utils.DeploymentLabelValue)
						if utils.DeploymentLabelKey != "" && utils.DeploymentLabelValue != "" {
							fmt.Println("I'm in")
							if v, ok := labelsMap[utils.DeploymentLabelKey]; ok == true && v == utils.DeploymentLabelValue {
								fmt.Println("Deployment with label ",utils.DeploymentLabelKey,":",utils.DeploymentLabelValue," modified")
								return true
							} else {
								fmt.Println("label not ",utils.DeploymentLabelKey,":",utils.DeploymentLabelValue)
								return false
							}
						}
						return false
					},

				},
			)*/

		//questo è un esempio di watch in cui il reconcile viene chiamato passandogli un altro N&N
		errWatch := c.Watch(
			&source.Kind{Type: &appsv1.Deployment{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
					//fmt.Println("a.Meta.GetName() :", a.Meta.GetName(), ", a.Meta.GetNameSpace() :", a.Meta.GetNamespace())

					return []reconcile.Request{
						{NamespacedName: types.NamespacedName{
							Name:      "firewall-sample",
							Namespace: "default",
						}},
					}
				}),
			},
		)
		if errWatch != nil {
			fmt.Println("watch non settato")
		} else {
			fmt.Println("watch settato")
		}
	}
	return err
}
