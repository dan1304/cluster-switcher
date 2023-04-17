import os
import subprocess
import time
import sys
from sys import exit
import pwinput
import base64

import logging

from dotenv import dotenv_values
env_vars = dotenv_values(os.path.expanduser('~/.setcluster_env'))

logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)

console_handler = logging.StreamHandler()
console_handler.setLevel(logging.INFO)

formatter = logging.Formatter()
console_handler.setFormatter(formatter)

logger.addHandler(console_handler)

def user_configure():
    default_okd_host = "https://api.c-th1n.ascendmoney.io:6443"
    default_ocp_host = "https://api.a-th1n.ascendmoney.io:6443"
    default_vpn_timeout = 80
    default_ocp_vpn = "thp-vpncen_nonprod-admin_v1"
    default_eks_vpn = "centralize-cloudops"

    ENV_OKD_HOST = input(f"OKD HOST (default: {default_okd_host}) ") or default_okd_host
    ENV_OCP_HOST = input(f"OCP HOST (default: {default_ocp_host}) ") or default_ocp_host
    ENV_OCP_VPN = input(f"OCP VPN (default: {default_ocp_vpn}) ") or str(default_ocp_vpn)
    ENV_EKS_VPN = input(f"EKS VPN (default: {default_eks_vpn}) ") or str(default_eks_vpn)
    ENV_VPN_TIMEOUT = input("VPN TIMEOUT (in seconds, default: 80) ") or str(default_vpn_timeout)
    ENV_OC_USERNAME = input("OC USERNAME (your LDAP username): ")
    ENV_OC_PWD_RAW =  pwinput.pwinput(prompt='OC PASSWORD (your LDAP password): ', mask='*') 
    ENV_OC_PWD_ENCODED = base64.b64encode(ENV_OC_PWD_RAW.encode('utf-8')).decode('utf-8')

    subprocess.run(f"echo ENV_OKD_HOST={ENV_OKD_HOST} > ~/.setcluster_env ", shell=True)
    subprocess.run(f"echo ENV_OCP_HOST={ENV_OCP_HOST} >> ~/.setcluster_env ", shell=True)
    subprocess.run(f"echo ENV_OC_USERNAME={ENV_OC_USERNAME} >> ~/.setcluster_env ", shell=True)
    subprocess.run(f"echo ENV_VPN_TIMEOUT={ENV_VPN_TIMEOUT} >> ~/.setcluster_env ", shell=True)    
    subprocess.run(f"echo ENV_OCP_VPN={ENV_OCP_VPN} >> ~/.setcluster_env ", shell=True)
    subprocess.run(f"echo ENV_EKS_VPN={ENV_EKS_VPN} >> ~/.setcluster_env ", shell=True)
    subprocess.run(f"echo ENV_OC_PWD_ENCODED={ENV_OC_PWD_ENCODED} >> ~/.setcluster_env ", shell=True)



if len(sys.argv) > 1:
    if sys.argv[1] == "help":
        print("""
Desc: Easy cluster switching  utility
Usage:   
    Init configuration:     setcluster configure
    Access cluster:         setcluster {cluster_name}
    Example, to access okd: setcluster okd
    Available clusters:
                            okd => OKD cluster (dev/qa/hotfix/sandbox)
                            ocp => OCP cluster (performance/staging)
                            eks => EKS cluster (qa/performance)
        """)
        exit()

    if sys.argv[1] == "configure":
        try:
            user_configure()
            logger.info("=>> Done configuration. Please run 'setcluster help' to see usage")
        except KeyboardInterrupt:
            logger.critical("Cancelled")
        exit()

    elif str(sys.argv[1]) not in ["okd", "eks", "ocp"]: 
        logger.info(f"Error! No cluster or command '{sys.argv[1]}' available. Please run 'setcluster help' to see the usage.")
        exit()
else:
    exit("Error! Please run 'setcluster help' to see the usage.")

choice_env = sys.argv[1] 
logger.info(f"Accessing to {choice_env.upper()} cluster...")
ENV_OC_PWD_ENCODED = env_vars['ENV_OC_PWD_ENCODED']
ENV_OC_PWD_DECODED = base64.b64decode(ENV_OC_PWD_ENCODED).decode('utf-8')
ENV_OC_USERNAME = env_vars['ENV_OC_USERNAME']


def connect_vpn(vpn_name):
    subprocess.run(["osascript", "-e", "tell application \"/Applications/Tunnelblick.app\"", "-e", "disconnect all",  "-e", "end tell"], stdout=subprocess.DEVNULL)
    time.sleep(3)
    subprocess.run(["osascript","-e", "tell application \"/Applications/Tunnelblick.app\"", "-e", f"connect \"{vpn_name}\"", "-e", "end tell"], stdout=subprocess.DEVNULL)


def verify_vpn_status(vpn_name):
    vpn_status = subprocess.run(["osascript", "-e", "tell application \"/Applications/Tunnelblick.app\"", 
                                "-e", f"get state of configurations where name = \"{vpn_name}\"", "-e", "end tell"],
                                stdout=subprocess.PIPE)
    return "CONNECTED" in str(vpn_status)


def login_to_openshift(openshift_host):
    subprocess.run(["oc","login", openshift_host, f"-u={ENV_OC_USERNAME}", f"-p={ENV_OC_PWD_DECODED}", "--insecure-skip-tls-verify=false"])


def login_to_eks(hostname=""):
    subprocess.run(['cloudopscli', 'login'])


def connect_and_login(vpn_name, hostname, login_func):
    if verify_vpn_status(vpn_name):
        login_func(hostname)
    else:
        connect_vpn(vpn_name)
        timeout = env_vars['ENV_VPN_TIMEOUT']
        start_time = time.time()
        while True:
            if verify_vpn_status(vpn_name):
                login_func(hostname)
                break
            elif time.time() - start_time >= int(timeout):
                logger.info("Timeout reached, unable to connect to VPN")
                break
            time.sleep(1)


def switch_choice_env(choice_env):
    switcher_hostname = {
        "ocp": env_vars['ENV_OCP_HOST'],
        "okd": env_vars['ENV_OKD_HOST'],
        "eks": ""
    }
    switcher_vpn_name = {
        "ocp": os.environ.get("ENV_OCP_VPN"),
        "okd": os.environ.get("ENV_OCP_VPN"),
        "eks": os.environ.get("ENV_EKS_VPN")
    }

    hostname = switcher_hostname.get(choice_env, "Invalid choice")
    vpn_name = switcher_vpn_name.get(choice_env, "Invalid choice")

    if  choice_env in ["okd", "ocp"]:
        connect_and_login(vpn_name, hostname, login_to_openshift)
    else:
        connect_and_login(vpn_name, hostname, login_to_eks)


if __name__ == '__main__':
    try:
        switch_choice_env(choice_env)
    except KeyboardInterrupt:
        logger.critical("Cancelled")
