#!/bin/sh

MINIMAL_ROOT=/home/user/work/minimal
TMP_ROOTFS=$MINIMAL_ROOT/src/work/tmp_rootfs

LDFLAGS="-L$TMP_ROOTFS/lib -L$TMP_ROOTFS/usr/lib" \
  CFLAGS="-I$TMP_ROOTFS/include -I$TMP_ROOTFS/usr/include" \
  PKG_CONFIG_SYSROOT_DIR=${TMP_ROOTFS} \
  PKG_CONFIG_ALLOW_SYSTEM_CFLAGS=1 \
  PKG_CONFIG_ALLOW_SYSTEM_LIBS=1 \
  PKG_CONFIG_PATH=$TMP_ROOTFS/lib/pkgconfig:$TMP_ROOTFS/usr/lib/pkgconfig \
  PKG_CONFIG_LIBDIR=$TMP_ROOTFS/lib/pkgconfig:$TMP_ROOTFS/usr/lib/pkgconfig \
  CGO_CFLAGS="-I${TMP_ROOTFS}/include -I${TMP_ROOTFS}/usr/include" \
  CGO_LDFLAGS="-L ${TMP_ROOTFS}/lib -L ${TMP_ROOTFS}/usr/lib" \
  go build -buildvcs=false .