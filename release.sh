#!/usr/bin/env bash

version="$(git describe)"
git clean -xdf


_build_release()
{
    exe="$1"
    builddir="$2"
    for os in linux freebsd ; do
        for arch in amd64 arm ppc64 ; do
            export XD=$builddir/$exe-$os-$arch
            GOOS=$os GOARCH=$arch make clean $XD && gpg --sign --detach $XD
        done
    done
    export XD=$builddir/$exe-darwin
    GOOS=darwin GOARCH=amd64 make clean $XD && gpg --sign --detach $XD
    export XD=$builddir/$exe-windows.exe
    GOOS=windows GOARCH=amd64 make clean $XD && gpg --sign --detach $XD
}


export GIT_VERSION=""
build=XD-$version
mkdir -p $build
# build i2p version
export LOKINET=0
_build_release XD-i2p-$version $build
# build lokinet version
export LOKINET=1
_build_release XD-lokinet-$version $build

# verify sigs and makes hashes
for sig in $build/*.sig ; do
    gpg --verify $sig && b2sum -b $(echo $sig | sed s/\\.sig//) >> $build/HASHES.txt
done

# check hashes
b2sum -c $build/HASHES.txt || exit 1

rm -f $build/README.txt
echo "To verify the integrity of XD $version use:" >> $build/README.txt 
echo "gpg --verify XD-$version.tar.xz.sig && tar -xJvf XD-$version.tar.xz && b2sum -c $build/HASHES.txt" >> $build/README.txt
echo "" >> $build/README.txt
echo "release hashes:" >> $build/README.txt
echo "" >> $build/README.txt
cat $build/HASHES.txt >> $build/README.txt

gpg --clearsign --detach $build/README.txt
mv $build/README.txt.asc $build/README.txt

# make release tarball
tar -cJvf XD-$version.tar.xz $build
gpg --sign --detach XD-$version.tar.xz

# verify sig and upload release
gpg --verify XD-$version.tar.xz.sig && gh release create --notes "XD $version" -R majestrate/XD -F $build/README.txt $version XD-$version.tar.xz{,.sig}

