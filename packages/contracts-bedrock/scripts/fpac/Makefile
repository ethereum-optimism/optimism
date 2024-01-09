# ENV
monorepo_root := "../../../.."
deploy_config_path := "../../deploy-config"
tmp := $(shell mktemp)

# Help menu
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make cannon-prestate chain=<chain>"
	@echo "  make deploy-fresh chain=<chain> args=<args>"
	@echo "  make upgrade-game-impl chain=<chain> dgf=<dgf> vm=<vm> args=<args>"
	@echo "  make update-init-bond chain=<chain> dgf=<dgf> game-type=<game-type> new-init-bond=<new-init-bond> args=<args>"

# Generate the cannon prestate, and tar the `op-program` + `cannon` binaries + prestate data used to generate it.
.PHONY: cannon-prestate
cannon-prestate:
	cd $(monorepo_root) && make cannon-prestate
	@mkdir -p prestate-artifacts
	@cp -r $(monorepo_root)/cannon/bin/** prestate-artifacts/
	@cp -r $(monorepo_root)/op-program/bin/** prestate-artifacts/
	tar -czvf prestate-artifacts.tar.gz prestate-artifacts
	@jq ".faultGameAbsolutePrestate = $$(cat prestate-artifacts/prestate-proof.json | jq .pre)" "$(deploy_config_path)/$(chain).json" > $(tmp) && mv $(tmp) "$(deploy_config_path)/$(chain).json"
	@echo "-------------------------------------------------------------------------------------------------------------"
	@echo "-> Archive of prestate artifacts available at prestate-artifacts.tar.gz"
	@echo "-> Set the absolute prestate to $$(cat prestate-artifacts/prestate-proof.json | jq .pre) in the $(chain) deploy config."
	@rm -rf prestate-artifacts

# Deploy a fresh version of the FPAC contracts. Pass `--broadcast` to send to the network.
.PHONY: deploy-fresh
deploy-fresh:
	forge script FPACOPS.sol --sig "deployFPAC()" --chain $(chain) -vvv $(args)

# Upgrades a dispute game implementation in the deployed `DisputeGameFactory` contract for the configured chain. Pass `--broadcast` to send to the network.
.PHONY: upgrade-game-impl
upgrade-game-impl:
	forge script FPACOPS.sol --sig "upgradeGameImpl(address,address)" --chain $(chain) -vvv $(dgf) $(vm) $(args)

# Upgrades a dispute game's init bond in the deployed `DisputeGameFactory` contract for the configured chain. Pass `--broadcast` to send to the network.
.PHONY: update-init-bond
update-init-bond:
	forge script FPACOPS.sol --sig "updateInitBond(address,uint8,uint256)" --chain $(chain) -vvv $(dgf) $(game-type) $(new-init-bond) $(args)
