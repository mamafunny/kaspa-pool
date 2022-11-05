# you need to go about exporting your keys.json from kaspawallet
# this command will create a configmap based on the contents 
# you should lock down this configmap and guard it with your life :)
# also 100% delete the keys.json file after you run this script
kubectl create configmap -n wallet-keys keys --from-file=keys.json