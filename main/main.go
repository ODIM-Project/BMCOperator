//(C) Copyright [2022] Hewlett Packard Enterprise Development LP
//
//Licensed under the Apache License, Version 2.0 (the "License"); you may
//not use this file except in compliance with the License. You may obtain
//a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//License for the specific language governing permissions and limitations
// under the License.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	bios "github.com/ODIM-Project/BMCOperator/controllers/bios"
	bmc "github.com/ODIM-Project/BMCOperator/controllers/bmc"
	boot "github.com/ODIM-Project/BMCOperator/controllers/boot"
	configuration "github.com/ODIM-Project/BMCOperator/controllers/config"
	eventsubscription "github.com/ODIM-Project/BMCOperator/controllers/eventsubscription"
	firmware "github.com/ODIM-Project/BMCOperator/controllers/firmware"
	odim "github.com/ODIM-Project/BMCOperator/controllers/odim"
	pollData "github.com/ODIM-Project/BMCOperator/controllers/pollData"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	volume "github.com/ODIM-Project/BMCOperator/controllers/volume"
	"github.com/ODIM-Project/BMCOperator/logs"

	"github.com/sirupsen/logrus"
	//+kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(infraiov1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	err := configuration.SetConfiguration()
	if err != nil {
		logs.Log.Fatal("unable to get config values" + err.Error())
	}
	hostName := os.Getenv("HOST_NAME")
	podName := os.Getenv("POD_NAME")
	pid := os.Getpid()
	logs.Adorn(logrus.Fields{
		"host":   hostName,
		"procid": podName + fmt.Sprintf("_%d", pid),
	})
	log := logs.Log
	log.Logger.SetLevel(configuration.Data.LogLevel)
	log.Logger.SetOutput(os.Stdout)
	logs.SetLogFormat(configuration.Data.LogFormat)
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", fmt.Sprintf(":%s", configuration.Data.MetricPort), "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", fmt.Sprintf(":%s", configuration.Data.HealthPort), "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	flag.Parse()

	// ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "a247721e.odimra",
	})

	cfg, _ := config.GetConfig()
	c, _ := client.New(cfg, client.Options{})
	//getting public private keys for encryption/decryption of passwords
	err = updateKeys(configuration.Data.SecretName, configuration.Data.Namespace, c)
	if err != nil {
		logs.Log.Fatal("unable to get public private keys" + err.Error())
	}
	if err = (&odim.OdimReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logs.Log.Fatal("unable to create odim controller" + err.Error())
	}
	if err = (&bmc.BmcReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logs.Log.Fatal("unable to create controller" + err.Error())
	}
	if err = (&bios.BiosSettingReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logs.Log.Fatal("unable to create controller" + err.Error())
	}
	if err = (&boot.BootOrderSettingsReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logs.Log.Fatal("unable to create controller" + err.Error())
	}
	if err = (&volume.VolumeReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Fatal("unable to create controller" + err.Error())
	}
	if err = (&firmware.FirmwareReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Fatal("unable to create controller" + err.Error())
	}
	if err = (&eventsubscription.EventsubscriptionReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Fatal("unable to create controller" + err.Error())
	}
	//+kubebuilder:scaffold:builder

	addIndex(mgr)

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logs.Log.Fatal("unable to set up health check" + err.Error())
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logs.Log.Fatal("unable to set up ready check" + err.Error())
	}
	errChan := make(chan error)
	go configuration.TrackConfigListener(errChan)
	logs.Log.Info("starting manager")
	go pollData.PollDetails(mgr)

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logs.Log.Fatal("problem running manager" + err.Error())
	}
}

func updateKeys(secretName, namespace string, client client.Client) error {
	keysSecret := &corev1.Secret{}
	encryptedPublicKey, encryptedPrivateKey := bmc.GetEncryptedPemKeysFromSecret(context.TODO(), keysSecret, secretName, namespace, client)
	privateKey, publicKey := bmc.GetPemKeys(context.TODO(), encryptedPublicKey, encryptedPrivateKey)
	if privateKey == nil || publicKey == nil {
		return errors.New("could not find public/private keys")
	}
	utils.PrivateKey = privateKey
	utils.PublicKey = publicKey
	utils.RootCA = keysSecret.Data["rootCACert"]
	return nil
}

// addIndex is used to create a index to search object based on given field
// r.Client.List() will work when we add the index here before using it
func addIndex(mgr manager.Manager) {
	cache := mgr.GetCache()
	bmcIPFunc := func(obj client.Object) []string {
		return []string{obj.(*infraiov1.Bmc).Spec.BmcDetails.Address}
	}

	if err := cache.IndexField(context.Background(), &infraiov1.Bmc{}, "spec.bmc.address", bmcIPFunc); err != nil {
		panic(err)
	}
	serialNoFunc := func(obj client.Object) []string {
		return []string{obj.(*infraiov1.Bmc).Status.SerialNumber}
	}

	if err := cache.IndexField(context.Background(), &infraiov1.Bmc{}, "status.serialNumber", serialNoFunc); err != nil {
		panic(err)
	}

	systemIDFunc := func(obj client.Object) []string {
		return []string{obj.(*infraiov1.Bmc).Status.BmcSystemID}
	}

	if err := cache.IndexField(context.Background(), &infraiov1.Bmc{}, "status.bmcSystemId", systemIDFunc); err != nil {
		panic(err)
	}

	metadataFunc := func(obj client.Object) []string {
		return []string{obj.(*infraiov1.Bmc).ObjectMeta.Name}
	}

	if err := cache.IndexField(context.Background(), &infraiov1.Bmc{}, "metadata.name", metadataFunc); err != nil {
		panic(err)
	}

	biosSchemaFunc := func(obj client.Object) []string {
		return []string{obj.(*infraiov1.BiosSchemaRegistry).Name}
	}

	if err := cache.IndexField(context.Background(), &infraiov1.BiosSchemaRegistry{}, "metadata.name", biosSchemaFunc); err != nil {
		panic(err)
	}

	biosFunc := func(obj client.Object) []string {
		return []string{obj.(*infraiov1.BiosSetting).ObjectMeta.Name}
	}

	if err := cache.IndexField(context.Background(), &infraiov1.BiosSetting{}, "metadata.name", biosFunc); err != nil {
		panic(err)
	}

	bootFunc := func(obj client.Object) []string {
		return []string{obj.(*infraiov1.BootOrderSetting).ObjectMeta.Name}
	}

	if err := cache.IndexField(context.Background(), &infraiov1.BootOrderSetting{}, "metadata.name", bootFunc); err != nil {
		panic(err)
	}

	odimNameFunc := func(obj client.Object) []string {
		return []string{obj.(*infraiov1.Odim).ObjectMeta.Name}
	}

	if err := cache.IndexField(context.Background(), &infraiov1.Odim{}, "metadata.name", odimNameFunc); err != nil {
		panic(err)
	}

	firmwareNameFunc := func(obj client.Object) []string {

		return []string{obj.(*infraiov1.Firmware).ObjectMeta.Name}

	}

	if err := cache.IndexField(context.Background(), &infraiov1.Firmware{}, "metadata.name", firmwareNameFunc); err != nil {

		panic(err)

	}

	eventSubscriptionIDFunc := func(obj client.Object) []string {
		return []string{obj.(*infraiov1.Eventsubscription).Status.ID}
	}

	if err := cache.IndexField(context.Background(), &infraiov1.Eventsubscription{}, "status.eventSubscriptionID", eventSubscriptionIDFunc); err != nil {
		panic(err)
	}

	eventSubscriptionObjNameFunc := func(obj client.Object) []string {
		return []string{obj.(*infraiov1.Eventsubscription).ObjectMeta.Name}
	}
	if err := cache.IndexField(context.Background(), &infraiov1.Eventsubscription{}, "metadata.name", eventSubscriptionObjNameFunc); err != nil {
		panic(err)
	}

	eventsMessageFunc := func(obj client.Object) []string {
		return []string{obj.(*infraiov1.EventsMessageRegistry).ObjectMeta.Name}
	}

	if err := cache.IndexField(context.Background(), &infraiov1.EventsMessageRegistry{}, "metadata.name", eventsMessageFunc); err != nil {
		panic(err)
	}
}
