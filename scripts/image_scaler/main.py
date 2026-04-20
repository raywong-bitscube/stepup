#!/usr/bin/env python3
"""
图片缩放工具：按指定比例等比例缩放图片（JPG/PNG）。

用法：
    python main.py --input input.jpg --scale 0.5 --output ./output
    python main.py --input /path/to/image.png --scale 0.8
    python main.py --input /path/to/image.jpg --scale 0.3

参数：
    --input   输入图片文件路径（JPG/PNG）
    --scale   缩放比例（0.1-1 之间的小数）
    --output  （可选）输出目录路径，缩放后的图片将保存到此目录（文件名与输入相同）
              如果未提供，则在输入文件同目录生成 {原文件名}_scaled.{扩展名}

日志将显示每个关键步骤，格式如 "1/7: 步骤描述"。
"""

from __future__ import annotations

import argparse
import logging
import sys
from pathlib import Path

try:
    from PIL import Image
except ImportError:
    print("错误：需要安装 Pillow 库。请运行：pip install Pillow")
    sys.exit(1)

log = logging.getLogger("image_scaler")


def parse_args() -> argparse.Namespace:
    """解析命令行参数"""
    parser = argparse.ArgumentParser(description="按比例缩放图片（JPG/PNG）")
    parser.add_argument(
        "--input",
        required=True,
        type=Path,
        help="输入图片文件路径（JPG/PNG）",
    )
    parser.add_argument(
        "--scale",
        required=True,
        type=float,
        help="缩放比例，必须是 0.1 到 1 之间的小数",
    )
    parser.add_argument(
        "--output",
        type=Path,
        help="（可选）输出目录路径，缩放后的图片将保存到此目录。如果未提供，则在输入文件同目录生成 {原文件名}_scaled.{扩展名}",
    )
    return parser.parse_args()


def validate_args(args: argparse.Namespace) -> None:
    """验证输入参数"""
    log.info("1/7: 验证输入参数")
    
    # 检查输入文件
    if not args.input.exists():
        raise FileNotFoundError(f"输入文件不存在：{args.input}")
    if not args.input.is_file():
        raise ValueError(f"输入路径不是文件：{args.input}")
    if args.input.suffix.lower() not in ('.jpg', '.jpeg', '.png'):
        raise ValueError(f"输入文件必须是 JPG 或 PNG 格式（.jpg/.jpeg/.png）：{args.input}")
    
    # 检查缩放比例
    if not 0.1 <= args.scale <= 1:
        raise ValueError(f"缩放比例必须在 0.1 到 1 之间，当前值：{args.scale}")
    
    log.info(f"  输入文件：{args.input}")
    log.info(f"  缩放比例：{args.scale}")
    if args.output:
        log.info(f"  输出目录：{args.output}")
    else:
        log.info("  输出目录：未指定，将在输入文件同目录生成缩放版本")


def determine_output_path(input_path: Path, output_dir: Path | None) -> Path:
    """确定输出文件路径并确保目录存在"""
    log.info("2/7: 确定输出路径")
    
    if output_dir:
        # 如果提供了 --output，则输出到指定目录，保持原文件名
        output_dir.mkdir(parents=True, exist_ok=True)
        output_file = output_dir / input_path.name
        log.info(f"  输出目录：{output_dir}")
        log.info(f"  输出文件：{output_file}")
    else:
        # 如果未提供 --output，则在输入文件同目录生成 _scaled 版本
        output_file = input_path.parent / f"{input_path.stem}_scaled{input_path.suffix}"
        log.info(f"  输出到同目录：{output_file}")
    
    return output_file


def load_image(input_path: Path) -> Image.Image:
    """加载输入图片"""
    log.info("3/7: 加载输入图片")
    try:
        img = Image.open(input_path)
        log.info(f"  图片加载成功：{img.width} x {img.height} 像素，格式：{img.format}")
        return img
    except Exception as e:
        raise RuntimeError(f"无法加载图片：{e}")


def calculate_new_size(img: Image.Image, scale: float) -> tuple[int, int]:
    """计算新的图片尺寸"""
    log.info("4/7: 计算新尺寸")
    new_width = int(img.width * scale)
    new_height = int(img.height * scale)
    log.info(f"  原始尺寸：{img.width} x {img.height}")
    log.info(f"  新尺寸：{new_width} x {new_height}")
    return new_width, new_height


def resize_image(img: Image.Image, new_size: tuple[int, int]) -> Image.Image:
    """缩放图片"""
    log.info("5/7: 缩放图片")
    try:
        # 使用 LANCZOS 重采样过滤器（高质量）
        resized_img = img.resize(new_size, Image.Resampling.LANCZOS)
        log.info("  图片缩放完成")
        return resized_img
    except Exception as e:
        raise RuntimeError(f"缩放图片时出错：{e}")


def save_image(img: Image.Image, output_path: Path) -> None:
    """保存图片到文件"""
    log.info("6/7: 保存图片")
    try:
        # 根据文件扩展名决定保存格式
        suffix = output_path.suffix.lower()
        if suffix in ('.jpg', '.jpeg'):
            # JPG 格式，质量设为 95（高质量）
            img.save(output_path, 'JPEG', quality=95, optimize=True)
            format_name = 'JPEG'
        elif suffix == '.png':
            # PNG 格式
            img.save(output_path, 'PNG', optimize=True)
            format_name = 'PNG'
        else:
            # 默认使用 JPG
            img.save(output_path, 'JPEG', quality=95, optimize=True)
            format_name = 'JPEG'
        
        log.info(f"  图片已保存：{output_path}（格式：{format_name}）")
    except Exception as e:
        raise RuntimeError(f"保存图片时出错：{e}")


def main() -> int:
    """主函数"""
    # 设置日志格式
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s [%(levelname)s] %(message)s',
        datefmt='%Y-%m-%d %H:%M:%S'
    )
    
    try:
        # 解析参数
        args = parse_args()
        
        # 验证参数
        validate_args(args)
        
        # 确定输出文件路径
        output_file = determine_output_path(args.input, args.output)
        
        # 加载图片
        img = load_image(args.input)
        
        # 计算新尺寸
        new_size = calculate_new_size(img, args.scale)
        
        # 缩放图片
        resized_img = resize_image(img, new_size)
        
        # 保存图片
        save_image(resized_img, output_file)
        
        # 完成
        log.info("7/7: 任务完成")
        log.info(f"  输出文件：{output_file}")
        log.info(f"  文件大小：{output_file.stat().st_size / 1024:.2f} KB")
        
        return 0
        
    except FileNotFoundError as e:
        log.error(f"文件错误：{e}")
        return 1
    except ValueError as e:
        log.error(f"参数错误：{e}")
        return 1
    except RuntimeError as e:
        log.error(f"处理错误：{e}")
        return 1
    except KeyboardInterrupt:
        log.info("用户中断")
        return 130
    except Exception as e:
        log.error(f"未预期的错误：{e}")
        return 1


if __name__ == "__main__":
    sys.exit(main())