# Pastebin RNsZP3u9
import numpy  as np
from PIL import Image
#import matplotlib.pyplot as plt

def loadImageAsNumarray(fn, newSize=None, cropSize=None, newMode=None):
    src=Image.open(fn)
    if src.mode=='P':
        src=src.convert('RGB')

    if(newMode):
        src=src.convert(newMode)

    if cropSize:
        oldSize=srcImage.size
        origin=(oldSize[0]-cropSize[0])/2,(oldSize[1]-cropSize[1])/2,
        srcImage=srcImage.crop((origin[0],origin[1],origin[0]+cropSize[0],origin[1]+cropSize[1]))

    if(newSize):
        src=src.resize(newSize)
    (w,h)=src.size
    d= {
        'L':1,
        'LA':2,
        'RGB':3,
        'RGBA':4,
    }[ src.mode]
    str=src.tobytes()
    ar=np.reshape(np.frombuffer(str,np.uint8)&255,(h,w,d))
    return ar

def saveNumarrayAsImage(srcArray, destfile):
    data=srcArray.astype(np.uint8).tobytes()
    if(len(srcArray.shape))==2:
        (height,width)=srcArray.shape
        depth=1
    else:
        (height,width,depth)=srcArray.shape
    mode=['L','LA','RGB','RGBA'][depth-1]
    im=Image.frombytes(mode,(width,height),data)
    im.save(destfile)

s=loadImageAsNumarray("dirty_madonna.gif")

bc=s[0,0]
def near(x):
    return np.less(np.abs(x)@[1,1,1],8)

def ytrim(s):
    bri=s@[1,1,1]
    bd=np.abs(bri-bri[0,0])
    tops=np.maximum.reduce(bd,axis=1)
    nz=np.nonzero(tops)[0]
    return s[nz[0]:nz[-1]+1]
s=ytrim(s)
s=ytrim(s.transpose((1,0,2))).transpose((1,0,2))
print("trimmed to",s.shape)

px=s.reshape((-1,3))
print(len(set(tuple(x) for x in px)))

px=s*1.0
pxl=np.roll(px, 1,axis=1)
pxr=np.roll(px,-1,axis=1)
mids=np.logical_and(
        near(px*2-pxl-pxr),
        1-near(pxl-pxr)
        )

def gc(s):
    v=set(tuple(x) for  x in s.reshape((-1,3)))
    v=list(v)
    v.sort()
    return v

allc=gc(px)

p1=gc(pxl[:,2:318][mids[:,2:318]])
p2=gc(pxr[:,2:318][mids[:,2:318]])
pures = list(set(p1+p2+[tuple(bc)]))
pures.sort(key=lambda x:np.array(x)@[1,2,1])

pures=np.array(pures)
K=len(pures)


blends=(pures.reshape((K,1,3))+pures.reshape((1,K,3)))/2
blends=blends.reshape((-1,3))



r=[]
toggles=[]
for c in allc:
    db=np.abs(c-blends)@[1,2,1]
    j0,j1=np.argsort(db)[:2]
    a,b=j0%K,j0//K
    c,d=j1%K,j1//K
    if a!=b:
        if a!=d or b!=c:
            print(a,b,c,d)
    toggles.append(a+b)

h,w=s.shape[:2]
ri=np.zeros((h,w),int)
for y in range(h):
    pi=0
    for x in range(w):
        ci=np.argmin((px[y,x]-allc)**2@[1,2,1])
        pi=toggles[ci]-pi
        ri[y,x]=pi

#plt.imshow(pures[ri]/255)
#plt.show()
saveNumarrayAsImage(pures[ri],"pure.png")
