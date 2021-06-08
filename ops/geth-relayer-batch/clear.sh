#!/bin/bash
path=/app/log
if [ ! -d $path ];
then
  mkdir -p $path
fi
num=`ls -l $path|grep "^-"|grep ".log"|wc -l`
if [ $num -lt 1 ] ;
then
  exit 0
fi
maxsize=$((1024*1024*50))
for file in $(ls $path/*.log)
do
  size=`ls -l $file|awk '{print $5}'`
  if [ $size -gt $maxsize ];
  then
    cat /dev/null>$file
  fi
done