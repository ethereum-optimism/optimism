import os
import logging

class MyEnv:

  def __init__(self,path):
    self.envFile = path
    self.envs = {}
 
  def SetEnvFile(self, filename) :
    self.envFile = filename
       
  def Save(self) :
    outf = open(self.envFile, "w")
    if not outf:
      print ("env file cannot be opened for write!")
    for k, v in self.envs.items() :
      x=v.replace('\\n', '')
      self.envs[k]=x
      outf.write(k+ "=" + x + "\n")
    outf.close()
   
  def Load(self) :
    inf = open(self.envFile, "r")
    if not inf:
      print ("env file cannot be opened for open!")
    for line in inf.readlines() :
      if line.startswith("#"):
        continue
      if len(line.strip())==0:
        continue
      k, v = line.split("=")
      self.envs[k] = v
    inf.close()
   
  def ClearAll(self) :
    self.envs.clear()
   
  def AddEnv(self, k, v) :
    self.envs[k] = v
   
  def RemoveEnv(self, k) :
    del self.envs[k]
   
  def PrintAll(self) :
    for k, v in self.envs.items():
      print ( k + "=" + v )
  
if __name__ == "__main__" :
  myEnv = MyEnv('')
  myEnv.SetEnvFile("/tmp/myenv.txt")
  myEnv.Load()
  myEnv.AddEnv("MYDIR", "/tmp/mydir")
  myEnv.AddEnv("MYDIR2", "/tmp/mydir2")
  myEnv.AddEnv("MYDIR3", "/tmp/mydir3")
  myEnv.Save()
  myEnv.PrintAll()