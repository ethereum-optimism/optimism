# Developing for OMGX on Windows

## Prequisites

- User folder can't have a space
	- A lot of the other programs we're using can't handle a space in paths.  
	So make sure there's no spaces for your users folder.
- [Windows Terminal](#windows-terminal)
- [Git](#git)
- [Docker](#docker)
- [NVM](#nvm)
- [Yarn](#yarn)

### Windows terminal

Windows terminal is a great substitute for terminal on MacOS/Linux. First download windows terminal from [here](https://www.microsoft.com/en-us/p/windows-terminal/9n0dx20hk701#activetab=pivot:overviewtab).  
Now that you have it installed we're going to install [Ubuntu](https://ubuntu.com/tutorials/ubuntu-on-windows#1-overview). Ubuntu creates a WSL2 environment for us to develop and it's on bash! Follow all the instructions from the link it's pretty straightforward to follow this guide. Once you have Ubuntu we're going to do everything from there not from powershell or cmd because they don't have access to all the bash commands. Make sure for all the other steps that you're running Windows Terminal as an administator as it'll make everything a lot easier. To do that right click on the app and then run as administrator.

### Git

Git should already be installed on your Ubuntu server but in case it's not follow [this](https://www.digitalocean.com/community/tutorials/how-to-install-git-on-ubuntu-18-04-quickstart) guide to get it up and running. 

### Docker

Install Docker from [here](https://docs.docker.com/docker-for-windows/install/). Follow through the instructions and restart your computer. If you run into this error:  
> `Hardware assisted virtualization and data execution protection must be enabled in the BIOS`

You're going to want to restart your computer and hold down **esc, f1, f2, f3, f4, f8 or delete** depending on your chip. It should tell you waht to hold when you're restarting to enter your bios settings. Then you're going to want to enable virtualization from here. This setting may be hidden under advanced &#8594; CPA configuration and then virtualization. Save this and then Docker should be up and running!

### NVM 

On your Ubuntu terminal **running as an admin** run the following command to instal cURL:
> `sudo apt-get install curl`

Then install nvm (node version manager):
> `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.35.3/install.sh | bash`

To verify installation, enter: `command -v nvm`. This should return 'nvm', if you receive 'command not found' or no response at all, close your current terminal, reopen it, and try again.

You can then use `nvm install [node-version]` to install whatever version of node you want. We recommend 14.17.3. Then use `nvm use [node-version]` to use that as your default. You can read more about NVM [here](https://github.com/nvm-sh/nvm). If you ran into any issues installing a lot of troubleshooting options can be found there as well.

### Yarn

Installing yarn is super easy with all the infrastructure we have use:

>`npm install --global yarn`

## All Done! 

You should have everything you need to follow the rest of the tutorial [here](https://github.com/omgnetwork/optimism/). Good luck and have fun developing on L2!


