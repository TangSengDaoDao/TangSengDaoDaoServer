
开源地址：

先用image2ascii将图片生成 ascii
https://github.com/qeesung/image2ascii


然后用ascii生成信息
https://github.com/dylanaraps/neofetch


// 生成

image2ascii -f docs/logo.jpg -r 0.08 > ascii_art.txt

neofetch --config ascii.conf --source ascii_art.txt > ascii_result.txt