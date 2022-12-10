set -e 
cd `dirname $0`
cd ..
rm -rf binary
mkdir binary

PHOENIX_BUILDER_DIR=".."
TIME_STAMP=$(date '+%m%d%H%M')
function get_hash(){
    fileName=$1
    outstr="$(md5sum $fileName)"
    hashStr="$(echo $outstr | cut -d' ' -f1)"
    echo "$hashStr"
}
git log -50 --color --reverse > ${PHOENIX_BUILDER_DIR}/omega/mainframe/update.log
make -C ${PHOENIX_BUILDER_DIR} clean 
make -C ${PHOENIX_BUILDER_DIR} linux-amd64 linux-arm64 windows-amd64 macos-amd64 macos-arm64 android-arm64 -j8
cp ${PHOENIX_BUILDER_DIR}/build/phoenixbuilder-linux-amd64 ./binary/fastbuilder-linux
cp ${PHOENIX_BUILDER_DIR}/build/phoenixbuilder-linux-arm64 ./binary/fastbuilder-linux-arm64
cp ${PHOENIX_BUILDER_DIR}/build/phoenixbuilder-windows-amd64.exe ./binary/fastbuilder-windows.exe
cp ${PHOENIX_BUILDER_DIR}/build/phoenixbuilder-macos-amd64 ./binary/fastbuilder-macos
cp ${PHOENIX_BUILDER_DIR}/build/phoenixbuilder-macos-arm64 ./binary/fastbuilder-macos-arm64
cp ${PHOENIX_BUILDER_DIR}/build/phoenixbuilder-android-arm64 ./binary/fastbuilder-android
echo $(get_hash ./binary/fastbuilder-linux) > ./binary/fastbuilder-linux.hash
echo $(get_hash ./binary/fastbuilder-linux-arm64) > ./binary/fastbuilder-linux.hash
echo $(get_hash ./binary/fastbuilder-windows.exe) > ./binary/fastbuilder-windows.hash
echo $(get_hash ./binary/fastbuilder-macos) > ./binary/fastbuilder-macos.hash
echo $(get_hash ./binary/fastbuilder-macos-arm64) > ./binary/fastbuilder-macos-arm64.hash
echo $(get_hash ./binary/fastbuilder-android) > ./binary/fastbuilder-android.hash
cat ./binary/*.hash > ./binary/all.hash
rm ${PHOENIX_BUILDER_DIR}/omega/mainframe/update.log
touch ${PHOENIX_BUILDER_DIR}/omega/mainframe/update.log

go run ./compressor/main.go -in "\
        ./binary/fastbuilder-linux,\
        ./binary/fastbuilder-linux-arm64,\
        ./binary/fastbuilder-windows.exe,\
        ./binary/fastbuilder-macos,\
        ./binary/fastbuilder-macos-arm64,\
        ./binary/fastbuilder-android\
    " -out "\
        ./binary/fastbuilder-linux.brotli,\
        ./binary/fastbuilder-linux-arm64.brotli, \
        ./binary/fastbuilder-windows.exe.brotli,\
        ./binary/fastbuilder-macos.brotli,\
         ./binary/fastbuilder-macos-arm64.brotli,\
        ./binary/fastbuilder-android.brotli\
    "

make -C ./launcher clean
make -C ./launcher all -j6
cp ./launcher/build/* ./binary
cp ./binary/launcher-linux ./binary/Linux版Omega启动器
cp ./binary/launcher-windows.exe ./binary/Windows版Omega启动器.exe
cp ./binary/launcher-macos ./binary/MacOS版Omega启动器
echo $(get_hash ./binary/launcher-linux-mcsm) > ./binary/launcher-linux-mcsm.hash
echo $(get_hash ./binary/launcher-linux) > ./binary/launcher-linux.hash
echo $(get_hash ./binary/launcher-macos) > ./binary/launcher-macos.hash
echo $(get_hash ./binary/launcher-android) > ./binary/launcher-android.hash

cp ./releaser/install.sh ./binary
cp ./releaser/dockerfile ./binary
cp ./releaser/docker_bootstrap.sh ./binary
git log -50 --reverse > ./binary/update.log

echo "$TIME_STAMP" >> ./binary/TIME_STAMP

rsync -avP --delete ./binary/* FBOmega:/var/www/omega-storage/binary/

cp ./releaser/install-cos1.sh ./binary/install.sh
./releaser/cos1 -d ./binary