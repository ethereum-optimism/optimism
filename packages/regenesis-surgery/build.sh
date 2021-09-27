#!/bin/bash
cd contracts/v3-core &&
  git apply ../../patches/v3-core.patch &&
  yarn install &&
  yarn compile &&
  git apply -R ../../patches/v3-core.patch &&
  cd ../../

cd contracts/v3-core-optimism &&
  git apply ../../patches/v3-core-optimism.patch &&
  yarn install &&
  yarn compile &&
  git apply -R ../../patches/v3-core-optimism.patch &&
  cd ../../
